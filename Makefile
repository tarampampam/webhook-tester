#!/usr/bin/make
# Makefile readme (ru): <http://linux.yaroslavl.ru/docs/prog/gnu_make_3-79_russian_manual.html>
# Makefile readme (en): <https://www.gnu.org/software/make/manual/html_node/index.html#SEC_Contents>

cwd = $(shell pwd)

SHELL = /bin/sh
LDFLAGS = "-s -w -X webhook-tester/version.version=$(shell git rev-parse HEAD)"

DOCKER_BIN = $(shell command -v docker 2> /dev/null)
DC_BIN = $(shell command -v docker-compose 2> /dev/null)
DC_RUN_ARGS = --rm --user "$(shell id -u):$(shell id -g)"
APP_NAME = $(notdir $(CURDIR))

.PHONY : help \
         image build fmt lint gotest test cover \
         clean
.DEFAULT_GOAL : help
.SILENT : lint gotest

help: ## Show this help
	@printf "\033[33m%s:\033[0m\n" 'Available commands'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[32m%-11s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

image: ## Build docker image with app
	$(DOCKER_BIN) build -f ./Dockerfile -t $(APP_NAME):local .
	$(DOCKER_BIN) run $(APP_NAME):local version
	@printf "\n   \e[30;42m %s \033[0m\n\n" 'Now you can use image like `docker run --rm $(APP_NAME):local ...`';

build: ## Build app binary file
	$(DC_BIN) run $(DC_RUN_ARGS) app go build -ldflags=$(LDFLAGS) .

fmt: ## Run source code formatter tools
	$(DC_BIN) run $(DC_RUN_ARGS) app sh -c 'GO111MODULE=off go get golang.org/x/tools/cmd/goimports && $$GOPATH/bin/goimports -d -w .'
	$(DC_BIN) run $(DC_RUN_ARGS) app gofmt -s -w -d .

lint: ## Run app linters
	$(DOCKER_BIN) run --rm -t -v "$(cwd):/app" -w /app golangci/golangci-lint:v1.24-alpine golangci-lint run

gotest: ## Run app tests
	$(DC_BIN) run $(DC_RUN_ARGS) app go test -v -race -timeout 5s ./... \
		&& printf "\n   \e[30;42m %s \033[0m\n\n" 'All tests passed!' \
		|| ( printf "\n   \e[39;41m %s \033[0m\n\n" 'Some tests fails!'; exit 1 )

test: lint gotest ## Run app tests and linters

cover: ## Run app tests with coverage report
	$(DC_BIN) run $(DC_RUN_ARGS) app sh -c 'go test -race -covermode=atomic -coverprofile /tmp/cp.out ./... && go tool cover -html=/tmp/cp.out -o ./coverage.html'
	-sensible-browser ./coverage.html && sleep 2 && rm -f ./coverage.html

shell: ## Start shell into container with golang
	$(DC_BIN) run $(DC_RUN_ARGS) app bash

clean: ## Make clean
	$(DC_BIN) down -v -t 1
	-$(DOCKER_BIN) rmi $(APP_NAME):local -f
