#!/usr/bin/make

DC_RUN_ARGS = --rm --user "$(shell id -u):$(shell id -g)"

.DEFAULT_GOAL : help
help: ## Show this help
	@printf "\033[33m%s:\033[0m\n" 'Available commands'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[32m%-16s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

shell: ## Start shell
	docker compose run $(DC_RUN_ARGS) app bash

web-shell: ## Start shell in web directory
	docker compose run $(DC_RUN_ARGS) -w '/src/web' app bash

generate: ## Run code generation
	docker compose run $(DC_RUN_ARGS) app go generate -skip readme ./...
	docker compose run $(DC_RUN_ARGS) app npm --prefix ./web run generate
	docker compose run $(DC_RUN_ARGS) app go generate -run readme ./...

node-build: ## Build the frontend
	docker compose run $(DC_RUN_ARGS) app npm --prefix ./web run build

node-fmt: ## Format frontend code
	docker compose run $(DC_RUN_ARGS) app npm --prefix ./web run fmt

lint: ## Run linters
	docker compose run $(DC_RUN_ARGS) app golangci-lint run

e2e: ## Run end-to-end tests
	docker compose run $(DC_RUN_ARGS) k6 run ./tests/k6/run.js

up: ## Start the application in watch mode
	#docker compose build
	docker compose kill app-http app-web-serve --remove-orphans 2>/dev/null || true
	docker compose up -d app-web-serve --wait # start the web dev server (vite)
	@printf "\n\t\033[33m%s\033[0m\n" "Open http://127.0.0.1:8080 in your browser to view the app in production mode (go server)"
	@printf "\t\033[33m%s\033[0m\n\n" "  or http://127.0.0.1:8081 to view the app web in development mode (vite, nodejs server)"
	docker compose up app-http

down: ## Stop the application
	docker compose down --remove-orphans
