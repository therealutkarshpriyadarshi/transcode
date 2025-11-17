.PHONY: help build test run-api run-worker docker-build docker-up docker-down migrate-up migrate-down clean

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build all binaries
	@echo "Building API..."
	@go build -o bin/api ./cmd/api
	@echo "Building Worker..."
	@go build -o bin/worker ./cmd/worker
	@echo "Build complete!"

test: ## Run tests
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Tests complete! Coverage report: coverage.html"

run-api: ## Run API server locally
	@echo "Starting API server..."
	@CONFIG_PATH=config.yaml go run cmd/api/main.go

run-worker: ## Run worker locally
	@echo "Starting worker..."
	@CONFIG_PATH=config.yaml go run cmd/worker/main.go

docker-build: ## Build Docker images
	@echo "Building Docker images..."
	@docker-compose build

docker-up: ## Start all services with Docker Compose
	@echo "Starting services..."
	@docker-compose up -d
	@echo "Services started!"
	@echo "API: http://localhost:8080"
	@echo "MinIO Console: http://localhost:9001"
	@echo "RabbitMQ Management: http://localhost:15672"

docker-down: ## Stop all services
	@echo "Stopping services..."
	@docker-compose down
	@echo "Services stopped!"

docker-logs: ## View logs from all services
	@docker-compose logs -f

migrate-up: ## Run database migrations
	@echo "Running migrations..."
	@docker-compose exec postgres psql -U postgres -d transcode -f /migrations/001_init_schema.up.sql
	@echo "Migrations complete!"

migrate-down: ## Rollback database migrations
	@echo "Rolling back migrations..."
	@docker-compose exec postgres psql -U postgres -d transcode -f /migrations/001_init_schema.down.sql
	@echo "Rollback complete!"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@echo "Clean complete!"

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "Dependencies downloaded!"

lint: ## Run linters
	@echo "Running linters..."
	@go fmt ./...
	@go vet ./...
	@echo "Linting complete!"
