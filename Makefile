.PHONY: all
all: format test build lint

.PHONY: generate
generate:
	rm -fr api/ pkg/apis/v1alpha1/api
	go tool ogen apis/v1alpha1/openapi.yaml -clean
	mv api pkg/apis/v1alpha1/

.PHONY: format
format:
	go fmt ./...

# Test commands based on test strategy
.PHONY: test
test:
	go test -v -parallel 4 ./...

# Version detection
VERSION ?= $(shell ./scripts/version.sh)
LDFLAGS := -X github.com/tacokumo/portal-api/pkg/version.Version=$(VERSION)

.PHONY: build
build:
	go build -ldflags "$(LDFLAGS)" -o bin/server ./cmd/server

.PHONY: lint
lint:
	golangci-lint run
