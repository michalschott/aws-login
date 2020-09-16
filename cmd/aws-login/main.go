package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sts"

	"github.com/michalschott/aws-login/pkg/random"

	log "github.com/sirupsen/logrus"
)

var (
	version = "unreleased"
	commit  = "git commit"
	date    = "2020"
)

func main() {
	// flag parse
	MfaValue := flag.String("mfa", "", "Value from MFA device")
	Duration := flag.Int64("duration", 3600, "Session duration")
	Debug := flag.Bool("debug", false, "Debug")
	Role := flag.String("role", "", "Role to assume")
	Account := flag.String("account", "", "Account number")
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

	log.Debugf("aws-login: %s, commit %s, build on %s", version, commit, date)

	// check if AWS_PROFILE is set
	if os.Getenv("AWS_PROFILE") == "" {
		log.Info("You have to set AWS_PROFILE environment variable first.")
		os.Exit(1)
	}

	// unset old/invalid/expired variables
	log.Debug("Unset variables is set to ", *NoUnset)
	if !*NoUnset {
		os.Unsetenv("AWS_ACCESS_KEY_ID")
		os.Unsetenv("AWS_SECRET_ACCESS_KEY")
		os.Unsetenv("AWS_SESSION_TOKEN")
	}

	// if MFA code is given, figure out MFA serial first
	MfaSerial := ""
	if *MfaValue != "" {
		iamSvc := iam.New(session.New())
		iamInput := &iam.ListMFADevicesInput{}

		result, err := iamSvc.ListMFADevices(iamInput)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case iam.ErrCodeNoSuchEntityException:
					log.Info(iam.ErrCodeNoSuchEntityException, aerr.Error())
				case iam.ErrCodeServiceFailureException:
					log.Info(iam.ErrCodeServiceFailureException, aerr.Error())
				default:
					log.Info(aerr.Error())
				}
			} else {
				log.Info(err.Error())
			}
			return
		}

		log.Debug(result)

		MfaSerial = *result.MFADevices[0].SerialNumber
	}

	stsSvc := sts.New(session.New())

	if *Role == "" {
		// just login with MFA

		// prepare input for GetSessionToken
		input := &sts.GetSessionTokenInput{}
		input.DurationSeconds = aws.Int64(*Duration)
		if *MfaValue != "" && MfaSerial != "" {
			input.SerialNumber = aws.String(MfaSerial)
			input.TokenCode = aws.String(*MfaValue)
		}
		log.Debug("Input request: ", input)

		// login
		result, err := stsSvc.GetSessionToken(input)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case sts.ErrCodeRegionDisabledException:
					log.Info(sts.ErrCodeRegionDisabledException, aerr.Error())
				default:
					log.Info(aerr.Error())
				}
			} else {
				log.Info(err.Error())
			}
			return
		}
		log.Debug(result)
		fmt.Printf("export %s=%v\n", "AWS_ACCESS_KEY_ID", *result.Credentials.AccessKeyId)
		fmt.Printf("export %s=%v\n", "AWS_SECRET_ACCESS_KEY", *result.Credentials.SecretAccessKey)
		fmt.Printf("export %s=%v\n", "AWS_SESSION_TOKEN", *result.Credentials.SessionToken)
	} else {
		// assume role

		// prepare input AssumeRole
		assumeRoleInput := &sts.AssumeRoleInput{}
		assumeRoleInput.DurationSeconds = aws.Int64(*Duration)
		if *MfaValue != "" && MfaSerial != "" {
			assumeRoleInput.SerialNumber = aws.String(MfaSerial)
			assumeRoleInput.TokenCode = aws.String(*MfaValue)
		}
		assumeRoleInput.RoleArn = aws.String("arn:aws:iam::" + *Account + ":role/" + *Role)
		if *RoleSessionName != "" {
			assumeRoleInput.RoleSessionName = aws.String(*RoleSessionName)
		} else {
			randomSessionName := random.RandomString(16)
			assumeRoleInput.RoleSessionName = &randomSessionName
		}
		log.Debug("Input request: ", assumeRoleInput)

		result, err := stsSvc.AssumeRole(assumeRoleInput)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				default:
					log.Info(aerr.Error())
				}
			} else {
				log.Info(aerr.Error())
			}
			return
		}
		log.Debug(result)
		fmt.Printf("export %s=%v\n", "AWS_ACCESS_KEY_ID", *result.Credentials.AccessKeyId)
		fmt.Printf("export %s=%v\n", "AWS_SECRET_ACCESS_KEY", *result.Credentials.SecretAccessKey)
		fmt.Printf("export %s=%v\n", "AWS_SESSION_TOKEN", *result.Credentials.SessionToken)
	}
}
