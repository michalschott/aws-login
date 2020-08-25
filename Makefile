.DEFAULT: all
.PHONY: all vet

VERSION=$(shell git symbolic-ref --short HEAD)-$(shell git rev-parse --short HEAD)

all: vet aws-login

vet:
	go vet -mod=vendor cmd/aws-login/*.go

clean:
	rm -f aws-login

aws-login: cmd/aws-login/*.go
	go build -mod=vendor -ldflags "-X main.version=$(VERSION)" -o $@ cmd/aws-login/*.go

