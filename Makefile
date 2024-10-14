#!/usr/bin/make

DC_RUN_ARGS = --rm --user "$(shell id -u):$(shell id -g)"

.DEFAULT_GOAL : help
help: ## Show this help
	@printf "\033[33m%s:\033[0m\n" 'Available commands'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[32m%-16s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

shell: ## Start shell
	docker compose run $(DC_RUN_ARGS) app bash

generate: ## Run code generation
	docker compose run $(DC_RUN_ARGS) app go generate ./...
	docker compose run $(DC_RUN_ARGS) app npm --prefix ./web run generate

node-build: ## Build the frontend
	docker compose run $(DC_RUN_ARGS) app npm --prefix ./web run build

node-fmt: ## Format frontend code
	docker compose run $(DC_RUN_ARGS) app npm --prefix ./web run fmt

lint: ## Run linters
	docker compose run $(DC_RUN_ARGS) app golangci-lint run

up: ## Start the application in watch mode
	docker compose kill daemon --remove-orphans 2>/dev/null || true
	docker compose up --detach --wait whoami httpbin
	docker compose up daemon

down: ## Stop the application
	docker compose down --remove-orphans
