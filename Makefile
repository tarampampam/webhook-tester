#!/usr/bin/make
# Makefile readme (en): <https://www.gnu.org/software/make/manual/html_node/index.html#SEC_Contents>

SHELL = /bin/sh
LDFLAGS = "-s -w -X github.com/tarampampam/webhook-tester/internal/pkg/version.version=$(shell git rev-parse HEAD)"

DC_RUN_ARGS = --rm --user "$(shell id -u):$(shell id -g)"
APP_NAME = $(notdir $(CURDIR))

.DEFAULT_GOAL : help

# This will output the help for each task. thanks to https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
help: ## Show this help
	@printf "\033[33m%s:\033[0m\n" 'Available commands'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[32m%-11s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

image: ## Build docker image with the app
	docker build -f ./Dockerfile -t $(APP_NAME):local .
	@printf "\n   \e[30;42m %s \033[0m\n\n" 'Now you can use image like `docker run --rm $(APP_NAME):local ...`'

frontend: node-install ## Build the frontend
	docker-compose run $(DC_RUN_ARGS) --no-deps node sh -c 'npm run gen && npm run build'

gen: ## Run code-generation
	docker-compose run $(DC_RUN_ARGS) --no-deps app go generate ./...

build: frontend gen ## Build app binary file
	docker-compose run $(DC_RUN_ARGS) -e "CGO_ENABLED=0" --no-deps app go build -trimpath -ldflags $(LDFLAGS) ./cmd/webhook-tester/

fmt: ## Run source code formatter tools
	docker-compose run $(DC_RUN_ARGS) --no-deps app sh -c 'go install golang.org/x/tools/cmd/goimports@latest && $$GOPATH/bin/goimports -d -w .'
	docker-compose run $(DC_RUN_ARGS) --no-deps app gofmt -s -w -d .
	docker-compose run $(DC_RUN_ARGS) --no-deps app go mod tidy

lint: ## Run app linters
	docker-compose run --rm --no-deps golint golangci-lint run
	docker-compose run --rm --no-deps node npm run lint

gotest: ## Run app tests
	docker-compose run $(DC_RUN_ARGS) --no-deps app go test -v -race -timeout 10s ./...

test: lint gotest ## Run app tests and linters

shell: ## Start shell inside golang environment
	docker-compose run $(DC_RUN_ARGS) app bash

node-install: ## Install node dependencies
	docker-compose run $(DC_RUN_ARGS) --no-deps node sh -c 'test -d ./node_modules || npm ci --no-audit --prefer-offline'

node-shell: ## Start shell inside node environment
	docker-compose run $(DC_RUN_ARGS) --no-deps node sh

up: ## Create and start containers
	docker-compose up --detach web
	@printf "\n   \e[30;42m %s \033[0m\n\n" 'Navigate your browser to ⇒ http://127.0.0.1:8080'

down: ## Stop all services
	docker-compose down -t 5 --remove-orphans

restart: down up ## Restart all containers

clean: ## Make clean
	docker-compose down -v -t 1
	-docker rmi $(APP_NAME):local -f
	-rm ./webhook-tester
