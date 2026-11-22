.PHONY: spec stop-spec stop build test

GITHUB_RUN ?= local
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_SHA ?= $(shell git rev-parse HEAD 2>/dev/null || echo unknown)
TAG ?= $(shell git describe --tags --exact-match 2>/dev/null || echo untagged)
VERSION_LDFLAGS := -X github.com/plannedwallet/internal/version.GitHubRun=$(GITHUB_RUN) -X github.com/plannedwallet/internal/version.BuildDate=$(BUILD_DATE) -X github.com/plannedwallet/internal/version.GitSHA=$(GIT_SHA) -X github.com/plannedwallet/internal/version.Tag=$(TAG)

build:
	go build -ldflags="$(VERSION_LDFLAGS)" -o server .

test:
	go test -v ./...

spec: stop-spec
	docker run --rm -d --name pw_spec -p 7979:8080 -v "$(shell pwd)":/tmp -e SWAGGER_JSON=/tmp/schema.yml docker.swagger.io/swaggerapi/swagger-ui

stop-spec:
	@docker stop pw_spec >/dev/null 2>&1 || true

stop: stop-spec
