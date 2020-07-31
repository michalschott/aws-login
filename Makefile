.DEFAULT: all
.PHONY: all vet

VERSION=$(shell git symbolic-ref --short HEAD)-$(shell git rev-parse --short HEAD)

all: vet cmd/aws-login

vet:
	go vet cmd/*.go

clean:
	rm -f cmd/aws-login

cmd/aws-login: cmd/*.go
	go build -ldflags "-X main.version=$(VERSION)" -o $@ cmd/*.go

