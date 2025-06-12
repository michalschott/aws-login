package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/michalschott/aws-login/pkg/random"

	log "github.com/sirupsen/logrus"
)

var (
	version = "unreleased"
	commit  = "git commit"
	date    = "2020"
)

type credentials struct {
	awsAccessKeyId     string
	awsSecretAccessKey string
	awsSessionToken    string
}

func (c *credentials) New(awsAccessKeyId string, awsSecretAccessKey string, awsSessionToken string) {
	c.awsAccessKeyId = awsAccessKeyId
	c.awsSecretAccessKey = awsSecretAccessKey
	c.awsSessionToken = awsSessionToken
}

func (c *credentials) Print(w io.Writer) {
	_, err := fmt.Fprintf(w, "export %s=%v\n", "AWS_ACCESS_KEY_ID", c.awsAccessKeyId)
	if err != nil {
		log.Debugf("Error writing to buffer")
		return
	}

	_, err = fmt.Fprintf(w, "export %s=%v\n", "AWS_SECRET_ACCESS_KEY", c.awsSecretAccessKey)
	if err != nil {
		log.Debugf("Error writing to buffer")
		return
	}

	_, err = fmt.Fprintf(w, "export %s=%v\n", "AWS_SESSION_TOKEN", c.awsSessionToken)
	if err != nil {
		log.Debugf("Error writing to buffer")
		return
	}
}

func main() {
	// flag parse
	MfaValue := flag.String("mfa", "", "Value from MFA device")
	Duration := flag.Int("duration", 3600, "Session duration")
	Debug := flag.Bool("debug", false, "Debug")
	Role := flag.String("role", "", "Role to assume")
	Account := flag.String("account", "", "Account number (if not set it will use sts.GetCallerIdentity call to figure out currently used accountID")
	RoleSessionName := flag.String("session-name", "", "Session name when assuming role")
	NoUnset := flag.Bool("nounset", false, "Should current AWS* env variables be unset before assuming new creds. Used in chain-assume scenarios.")
	flag.Parse()

	// logger configuration
	if *Debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
	log.SetFormatter(&log.TextFormatter{
		DisableColors: true,
	})
	log.SetFormatter(&log.JSONFormatter{})

	log.Debugf("aws-login: %s, commit %s, build on %s", version, commit, date)

	// check if AWS_PROFILE is set
	if os.Getenv("AWS_PROFILE") == "" {
		log.Info("AWS_PROFILE is not set, defaulting to 'default'.")
		err := os.Setenv("AWS_PROFILE", "default")
		if err != nil {
			log.Fatal(err)
		}
	}

	// unset old/invalid/expired variables
	log.Debug("Unset variables is set to ", *NoUnset)
	if !*NoUnset {
		envs := []string{
			"AWS_ACCESS_KEY_ID",
			"AWS_SECRET_ACCESS_KEY",
			"AWS_SESSION_TOKEN",
		}
		for _, v := range envs {
			err := os.Unsetenv(v)
			if err != nil {
				log.Info("Can not unset env var ", v)
			}
		}
	}

	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("eu-west-1"))
	if err != nil {
		log.Info(err)
	}

	stsSvc := sts.NewFromConfig(cfg)

	// if MFA code is given, figure out MFA serial first
	MfaSerial := ""
	if *MfaValue != "" {
		stsInput := &sts.GetCallerIdentityInput{}

		result, err := stsSvc.GetCallerIdentity(ctx, stsInput)
		if err != nil {
			log.Info(err.Error())
			return
		}

		MfaSerial = strings.Replace(aws.ToString(result.Arn), "user", "mfa", 1)
	}

	credentials := new(credentials)

	if *Role == "" {
		// just login with MFA

		// prepare input for GetSessionToken
		input := &sts.GetSessionTokenInput{}
		*input.DurationSeconds, err = random.IntToInt32(*Duration)
		if err != nil {
			log.Info(err.Error())
			return
		}
		if *MfaValue != "" && MfaSerial != "" {
			input.SerialNumber = aws.String(MfaSerial)
			input.TokenCode = aws.String(*MfaValue)
		}
		log.Debug("Input request: ", input)

		// login
		result, err := stsSvc.GetSessionToken(ctx, input)
		if err != nil {
			log.Info(err.Error())
			return
		}
		log.Debug(result)
		credentials.New(*result.Credentials.AccessKeyId, *result.Credentials.SecretAccessKey, *result.Credentials.SessionToken)
	} else {
		// assume role

		if *Account == "" {
			// get current account number
			result, err := stsSvc.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
			if err != nil {
				log.Info(err.Error())
				return
			}

			Account = result.Account
		}

		// prepare input AssumeRole
		assumeRoleInput := &sts.AssumeRoleInput{}
		*assumeRoleInput.DurationSeconds, err = random.IntToInt32(*Duration)
		if err != nil {
			log.Info(err.Error())
			return
		}
		if *MfaValue != "" && MfaSerial != "" {
			assumeRoleInput.SerialNumber = aws.String(MfaSerial)
			assumeRoleInput.TokenCode = aws.String(*MfaValue)
		}
		assumeRoleInput.RoleArn = aws.String("arn:aws:iam::" + *Account + ":role/" + *Role)
		if *RoleSessionName != "" {
			assumeRoleInput.RoleSessionName = aws.String(*RoleSessionName)
		} else {
			randomStringConfig := random.RandomStringConfig{
				Length:  16,
				Charset: "abcdefghijklmnopqrstuvwxyz" + "ABCDEFGHIJKLMNOPQRSTUVWXYZ" + "0123456789",
			}

			randomSessionName, err := randomStringConfig.New()
			if err != nil {
				log.Fatal("Can not generate sesssion name.")
			}

			assumeRoleInput.RoleSessionName = &randomSessionName
		}
		log.Debug("Input request: ", assumeRoleInput)

		result, err := stsSvc.AssumeRole(ctx, assumeRoleInput)
		if err != nil {
			log.Info(err.Error())
			return
		}
		log.Debug(result)
		credentials.New(*result.Credentials.AccessKeyId, *result.Credentials.SecretAccessKey, *result.Credentials.SessionToken)
	}

	credentials.Print(os.Stdout)
}
