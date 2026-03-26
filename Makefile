HOSTNAME=registry.terraform.io
NAMESPACE=ishosting
NAME=ishosting
BINARY=terraform-provider-${NAME}
VERSION=0.1.0
OS_ARCH=$(shell go env GOOS)_$(shell go env GOARCH)

default: install

build:
	go build -o ${BINARY}

install: build
	mkdir -p ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}
	mv ${BINARY} ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}/

test:
	go test ./... -v

testacc:
	TF_ACC=1 go test ./... -v

fmt:
	go fmt ./...

vet:
	go vet ./...

.PHONY: build install test testacc fmt vet
