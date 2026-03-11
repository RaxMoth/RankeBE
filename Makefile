.PHONY: help build run test clean docker-build docker-up docker-down migrate swagger

# Variables
APP_NAME=gin-rest-api
MAIN_PATH=./cmd/api
DOCKER_COMPOSE=docker-compose

help: ## Display this help message
	@echo "Available commands:"
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

test-coverage: test ## Run tests with coverage report
	go tool cover -html=coverage.out

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out

# Docker commands
docker-build: ## Build Docker image
	$(DOCKER_COMPOSE) build

docker-up: ## Start Docker containers
	$(DOCKER_COMPOSE) up -d

docker-down: ## Stop Docker containers
	$(DOCKER_COMPOSE) down

docker-logs: ## View Docker logs
	$(DOCKER_COMPOSE) logs -f

docker-restart: docker-down docker-up ## Restart Docker containers

docker-clean: ## Remove Docker containers and volumes
	$(DOCKER_COMPOSE) down -v

# Development
dev: ## Run in development mode with hot reload (requires air)
	air

# Swagger
swagger: ## Generate Swagger documentation
	swag init -g cmd/api/main.go -o ./docs

swagger-serve: ## Serve Swagger UI
	@echo "Opening Swagger UI at http://localhost:8080/swagger/index.html"
	@open http://localhost:8080/swagger/index.html || xdg-open http://localhost:8080/swagger/index.html

# Database migrations (requires golang-migrate)
migrate-create: ## Create a new migration (usage: make migrate-create name=migration_name)
	migrate create -ext sql -dir migrations -seq $(name)

migrate-up: ## Run database migrations
	migrate -path migrations -database "mysql://root:rootpassword@tcp(localhost:3306)/gin_rest_db" up

migrate-down: ## Rollback database migrations
	migrate -path migrations -database "mysql://root:rootpassword@tcp(localhost:3306)/gin_rest_db" down

migrate-force: ## Force migration version (usage: make migrate-force version=1)
	migrate -path migrations -database "mysql://root:rootpassword@tcp(localhost:3306)/gin_rest_db" force $(version)

# Local Development
local-db: ## Start MySQL database in Docker for local development
	@echo "Starting MySQL database locally..."
	@docker run -d \
		--name gin-rest-mysql-local \
		-e MYSQL_ROOT_PASSWORD=rootpassword \
		-e MYSQL_USER=gin_user \
		-e MYSQL_PASSWORD=secure_password_123 \
		-e MYSQL_DATABASE=gin_rest_db \
		-p 3306:3306 \
		--health-cmd="mysqladmin ping -h localhost" \
		--health-interval=10s \
		--health-timeout=5s \
		--health-retries=5 \
		mysql:8.0 2>/dev/null || echo "MySQL container already running or port in use"
	@echo "MySQL started on localhost:3306"
	@echo "Wait a few seconds for MySQL to be ready before running the app"

local-db-stop: ## Stop the local MySQL database
	@echo "Stopping MySQL database..."
	@docker stop gin-rest-mysql-local 2>/dev/null || echo "Container not running"
	@docker rm gin-rest-mysql-local 2>/dev/null || echo "Container already removed"

local-db-logs: ## View local MySQL logs
	@docker logs -f gin-rest-mysql-local

local-migrate: ## Run database migrations for local development
	@echo "Running database migrations..."
	migrate -path migrations -database "mysql://gin_user:secure_password_123@tcp(localhost:3306)/gin_rest_db" up

local-run: build ## Build and run the application locally with .env.local
	@echo "Starting application in development mode..."
	@export $$(cat .env.local | grep -v ^\# | xargs -0) && ./bin/$(APP_NAME)

local-setup: install local-db ## Setup local development environment (installs deps and starts MySQL)
	@echo "Local setup complete!"
	@echo "Next steps:"
	@echo "1. Wait for MySQL to be ready (few seconds)"
	@echo "2. Run: make local-migrate (to setup database schema)"
	@echo "3. Run: make local-run (to start the application)"

# Linting
lint: ## Run golangci-lint
	golangci-lint run

lint-fix: ## Run golangci-lint with auto-fix
	golangci-lint run --fix

# Format
fmt: ## Format code
	go fmt ./...
	gofmt -s -w .

# Security
security: ## Run security checks
	gosec ./...

# Generate
generate: ## Run go generate
	go generate ./...

# All-in-one commands
setup: install ## Initial project setup
	cp .env.example .env
	@echo "Please update .env with your configuration"

dev-setup: setup docker-up swagger ## Setup for local development
	@echo "Development environment ready!"
	@echo "API running at http://localhost:8080"
	@echo "Swagger docs at http://localhost:8080/swagger/index.html"
