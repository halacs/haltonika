DIST=dist
APPNAME=haltonika

GOLANGCILINT_VERSION=v2.8.0
GOSEC_VERSION=v2.22.11
VULNCHECK_VERSION=latest

VERSION ?= $(shell git describe --tags --dirty)
BUILD_DATE ?= $(shell date --rfc-email)

all: clean build

env:
	mkdir -p ${DIST}

clean:
	rm -rf ${DIST}

lint-env:
	( which gosec &>/dev/zero && gosec --version | grep -qs $(GOSEC_VERSION) ) || go install github.com/securego/gosec/v2/cmd/gosec@$(GOSEC_VERSION)
	( which golangci-lint &>/dev/zero && golangci-lint --version | grep -qs $(GOLANGCILINT_VERSION) ) || go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCILINT_VERSION)
	( which govulncheck &>/dev/zero ) || go install golang.org/x/vuln/cmd/govulncheck@$(VULNCHECK_VERSION)

lint: lint-env
	golangci-lint --timeout 10m run -v ./...
	gosec ./...
	govulncheck ./...

lint-fix: lint-env
	golangci-lint run -v --fix ./...

test: test-short
	go test ${VENDOR} ./...

test-short:
	go test ${VENDOR} -race -short

build: env
	CGO_ENABLED=0 go build -ldflags "-X 'github.com/halacs/haltonika/version.Version=${VERSION}' -X 'github.com/halacs/haltonika/version.BuildDate=${BUILD_DATE}'" -v -o ${DIST}/${APPNAME} .

