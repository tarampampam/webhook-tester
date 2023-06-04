#!/usr/bin/make
# Makefile readme (en): <https://www.gnu.org/software/make/manual/html_node/index.html#SEC_Contents>

SHELL = /bin/sh
LDFLAGS = "-s -w -X github.com/tarampampam/webhook-tester/internal/version.version=$(shell git rev-parse HEAD)"

DC_RUN_ARGS = --rm --user "$(shell id -u):$(shell id -g)"
APP_NAME = $(notdir $(CURDIR))

.DEFAULT_GOAL : help

# This will output the help for each task. thanks to https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
help: ## Show this help
	@printf "\033[33m%s:\033[0m\n" 'Available commands'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[32m%-16s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

image: ## Build docker image with the app
	docker build -f ./Dockerfile -t $(APP_NAME):local .
	docker images $(APP_NAME):local # print the info
	@printf "\n   \e[30;42m %s \033[0m\n\n" 'Now you can use image like `docker run --rm $(APP_NAME):local ...`'

fake-web-dist: # Is needed for the backend (embedding)
	test -d ./web/dist || ( mkdir ./web/dist && touch ./web/dist/index.html )

# Frontend stuff

node-install: ## Install node dependencies
	docker-compose run $(DC_RUN_ARGS) node sh -c 'test -d ./node_modules || npm ci --no-audit --prefer-offline'

node-generate: node-install ## Generate frontend assets
	docker-compose run $(DC_RUN_ARGS) node npm run generate

node-build: node-generate ## Build the frontend
	-rm -R ./web/dist
	docker-compose run $(DC_RUN_ARGS) node npm run build

node-lint: node-generate ## Lint the frontend
	docker-compose run $(DC_RUN_ARGS) node npm run lint

node-shell: ## Start shell inside node environment
	docker-compose run $(DC_RUN_ARGS) node sh

# Backend stuff

go-generate: ## Generate backend assets
	docker-compose run $(DC_RUN_ARGS) --no-deps go go generate ./...

go-build: node-build go-generate ## Build app binary file
	docker-compose run $(DC_RUN_ARGS) -e "CGO_ENABLED=0" --no-deps go go build -trimpath -ldflags $(LDFLAGS) ./cmd/webhook-tester/

go-test: fake-web-dist go-generate ## Run backend tests
	docker-compose run $(DC_RUN_ARGS) --no-deps go go test -v -race -timeout 10s ./...

go-lint: fake-web-dist go-generate ## Link the backend
	docker-compose run --rm golint golangci-lint run

go-fmt: fake-web-dist ## Run source code formatting tools
	docker-compose run $(DC_RUN_ARGS) --no-deps go gofmt -s -w -d .
	docker-compose run $(DC_RUN_ARGS) --no-deps go goimports -d -w .
	docker-compose run $(DC_RUN_ARGS) --no-deps go go mod tidy

go-shell: ## Start shell inside go environment
	docker-compose run $(DC_RUN_ARGS) go sh

# Overall stuff

e2e-test: ## Run integration (E2E) tests
	docker-compose run --rm hurl

test: go-lint go-test node-lint ## Run all tests

up: node-generate go-generate ## Start the app in the development mode
	docker-compose up --detach node-watch web

down: ## Stop the app
	docker-compose down --remove-orphans

restart: down up ## Restart all containers

clean: ## Make clean
	docker-compose down -v -t 1
	-docker rmi $(APP_NAME):local -f
	-rm -R ./webhook-tester ./web/node_modules ./web/dist
