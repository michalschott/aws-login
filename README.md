## Introduction

AWS login helper will help you setup session using aws-cli, no matter if you need to provide MFA token or assume role.

## Build

Clone locally and build with `make`. Move `aws-login` binary to your `PATH` ie. `mv aws-build ~/bin`

or

```
go get github.com/michalschott/aws-login/cmd/aws-login
```

## Usage

```
Usage of ./aws-login:
  -debug
    	Debug
  -duration int
    	Session duration (default 3600)
  -mfa string
    	Value from MFA device
  -role string
    	Role to assume
  -session-name string
    	Session name when assuming role
```

Simpliest way to export new temporary session variables is to execute:
```
eval $(cmd/aws-login)
```
