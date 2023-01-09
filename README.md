## Introduction

AWS login helper will help you setup session using aws-cli, no matter if you need to provide MFA token or assume role.

## Install

Either download binary from release, or you can build it yourself:

```
go install github.com/michalschott/aws-login/cmd/aws-login@latest
~/go/bin/aws-login
```

## Usage

```
Usage of aws-login:
  -account string
    	Account number (if not set it will use sts.GetCallerIdentity call to figure out currently used accountID
  -debug
    	Debug
  -duration int
    	Session duration (default 3600)
  -mfa string
    	Value from MFA device
  -nounset
    	Should current AWS* env variables be unset before assuming new creds. Used in chain-assume scenarios.
  -role string
    	Role to assume
  -session-name string
    	Session name when assuming role
```

Simpliest way to export new temporary session variables is to execute:
```
eval $(cmd/aws-login)
```
