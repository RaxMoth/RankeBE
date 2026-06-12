.PHONY: help install build run test test-integration clean sqlc migrate fmt vet lint dev-up dev-down dev-logs dev-reset

APP_NAME=ranke-server
MAIN_PATH=./cmd/server

# Default Postgres URL used by `make migrate` and `make test-integration`
# when nothing is set in the environment. Matches docker-compose.yml.
DEV_DATABASE_URL ?= postgres://ranke:ranke@localhost:5432/ranke?sslmode=disable

help: ## Display this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

install: ## Install dependencies
	go mod download
	go mod tidy

build: ## Build the application
	go build -o bin/$(APP_NAME) $(MAIN_PATH)

run: ## Run the application (uses .env)
	go run $(MAIN_PATH)

test: ## Run unit tests (skips anything needing TEST_DATABASE_URL)
	go test -race ./...

test-integration: ## Run all tests against the dev Postgres
	TEST_DATABASE_URL='$(DEV_DATABASE_URL)' go test -race -count=1 ./...

clean: ## Clean build artifacts
	rm -rf bin/ coverage.out

sqlc: ## Regenerate sqlc code from SQL queries
	sqlc generate

migrate: ## Apply all migrations to local PostgreSQL (requires psql)
	@for f in internal/db/migrations/*.sql; do \
	  echo "Applying $$f"; \
	  psql "$${DATABASE_URL:-$(DEV_DATABASE_URL)}" -v ON_ERROR_STOP=1 -f "$$f" || exit 1; \
	done

fmt: ## Format code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

lint: ## Run golangci-lint
	golangci-lint run

# ── docker-compose helpers ────────────────────────────────────────────
dev-up: ## docker compose up --build (Postgres + API)
	docker compose up --build

dev-down: ## docker compose down (keep data volume)
	docker compose down

dev-logs: ## Tail compose logs
	docker compose logs -f

dev-reset: ## Wipe the Postgres volume so migrations re-apply on next dev-up
	docker compose down -v
