.PHONY: help build run test clean sqlc migrate

APP_NAME=ranke-server
MAIN_PATH=./cmd/server

help: ## Display this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

install: ## Install dependencies
	go mod download
	go mod tidy

build: ## Build the application
	go build -o bin/$(APP_NAME) $(MAIN_PATH)

run: ## Run the application
	go run $(MAIN_PATH)

test: ## Run tests
	go test -v -race -coverprofile=coverage.out ./...

clean: ## Clean build artifacts
	rm -rf bin/ coverage.out

sqlc: ## Generate sqlc code from SQL queries
	sqlc generate

migrate: ## Apply migration to local PostgreSQL (requires psql)
	psql "$$DATABASE_URL" -f internal/db/migrations/001_init.sql

fmt: ## Format code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

lint: ## Run golangci-lint
	golangci-lint run
