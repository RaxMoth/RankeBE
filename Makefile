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

migrate: ## Apply all migrations to local PostgreSQL (requires psql)
	@for f in internal/db/migrations/*.sql; do \
	  echo "Applying $$f"; \
	  psql "$$DATABASE_URL" -v ON_ERROR_STOP=1 -f "$$f" || exit 1; \
	done

fmt: ## Format code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

lint: ## Run golangci-lint
	golangci-lint run
