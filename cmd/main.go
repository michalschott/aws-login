package main

import (
	"flag"
	"fmt"
	"math/rand"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sts"

	log "github.com/sirupsen/logrus"
)

const charset = "abcdefghijklmnopqrstuvwxyz" + "ABCDEFGHIJKLMNOPQRSTUVWXYZ" + "0123456789"

var (
	seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	version               = "unreleased"
)

func randomStringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func randomString(length int) string {
	return randomStringWithCharset(length, charset)
}

func main() {
	// flag parse
	MfaValue := flag.String("mfa", "", "Value from MFA device")
	Duration := flag.Int64("duration", 3600, "Session duration")
	Debug := flag.Bool("debug", false, "Debug")
	Role := flag.String("role", "", "Role to assume")
	RoleSessionName := flag.String("session-name", "", "Session name when assuming role")
	flag.Parse()

	log.Debugf("aws-login: %s", version)

	// logger configuration
	if *Debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
	log.SetFormatter(&log.TextFormatter{
		DisableColors: true,
	})

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
		assumeRoleInput.RoleArn = aws.String(*Role)
		if *RoleSessionName != "" {
			assumeRoleInput.RoleSessionName = aws.String(*RoleSessionName)
		} else {
			randomSessionName := randomString(16)
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
