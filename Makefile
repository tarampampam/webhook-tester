#!/usr/bin/make

.DEFAULT_GOAL : help
help: ## Show this help
	@printf "\033[33m%s:\033[0m\n" 'Available commands'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[32m%-16s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

install: ## Install dependencies
	@test -d ./web/node_modules || npm --prefix ./web install --no-audit
	@go mod download

gen: install ## Run code generation
	go generate -skip readme ./...  # generate all except readme
	npm --prefix ./web run generate # generate web code
	go generate -run readme ./...   # update readme

web-build: install ## Build the frontend
	npm --prefix ./web run build

build: web-build ## Build the application
	GOAMD64=v2 CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags "-s -w" ./cmd/webhook-tester/

fmt: install ## Apply code formatting
	npm --prefix ./web run fmt
	go fix ./...
	go fmt ./...
	golangci-lint run --fix --issues-exit-code 0 || true

lint: install ## Run linters
	golangci-lint run --fix ./...
	npm --prefix ./web run lint

test: install ## Run tests
	go test -v -race ./...
	npm --prefix ./web run test

up: install ## Start the application in development mode
	@go run ./cmd/webhook-tester/ --port 8081 & BACKEND_PID=$$!; \
	DEV_SERVER_PROXY_TO='http://localhost:8081' npm run --prefix ./web serve -- --port 8080 & FRONTEND_PID=$$!; \
	trap 'kill $$BACKEND_PID $$FRONTEND_PID' INT TERM EXIT; \
	printf "\n\t\033[1;7;33m %s \033[0m\n\n" "Press Ctrl+C to stop the development servers"; \
	wait
