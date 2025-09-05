.PHONY: help setup build test clean dev prod stop logs

# Default target
.DEFAULT_GOAL := help

# Variables
DOCKER_COMPOSE = docker-compose
GO = go
APP_NAME = indico-backend

help: ## Show this help message
	@echo "Indico Backend Makefile"
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

setup: ## Set up development environment
	@echo "Setting up development environment..."
	@cp -n .env.example .env || true
	@$(DOCKER_COMPOSE) up -d postgres postgres_test
	@echo "Waiting for PostgreSQL to be ready..."
	@sleep 5
	@$(GO) mod download
	@echo "Development environment ready!"

build: ## Build the application binary
	@echo "Building application..."
	@$(GO) build -o bin/server cmd/server/main.go
	@$(GO) build -o bin/seeder cmd/seeder/main.go
	@echo "Build complete!"

build-docker: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t $(APP_NAME) .
	@echo "Docker image built!"

test: ## Run tests
	@echo "Running tests..."
	@$(DOCKER_COMPOSE) up -d postgres_test
	@sleep 3
	@$(GO) test ./test/... -v

test-concurrent: ## Run concurrent order test specifically
	@echo "Running concurrent order test..."
	@$(DOCKER_COMPOSE) up -d postgres_test
	@sleep 3
	@$(GO) test ./test/... -run TestConcurrentOrders -v

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@$(DOCKER_COMPOSE) up -d postgres_test
	@sleep 3
	@$(GO) test ./test/... -coverprofile=coverage.out
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

lint: ## Run linter
	@echo "Running linter..."
	@golangci-lint run

fmt: ## Format code
	@echo "Formatting code..."
	@$(GO) fmt ./...

dev: ## Start development server
	@echo "Starting development server..."
	@$(DOCKER_COMPOSE) up -d postgres
	@sleep 3
	@$(GO) run cmd/server/main.go

seed: ## Seed test data
	@echo "Seeding test data..."
	@$(DOCKER_COMPOSE) up -d postgres
	@sleep 3
	@$(GO) run cmd/seeder/main.go

prod: ## Start production services
	@echo "Starting production services..."
	@$(DOCKER_COMPOSE) up -d
	@echo "Services started!"
	@echo "API: http://localhost:8080"
	@echo "Grafana: http://localhost:3000 (admin/admin)"
	@echo "Prometheus: http://localhost:9090"

stop: ## Stop all services
	@echo "Stopping services..."
	@$(DOCKER_COMPOSE) down

logs: ## Show application logs
	@$(DOCKER_COMPOSE) logs -f app

db-shell: ## Open database shell
	@$(DOCKER_COMPOSE) exec postgres psql -U postgres -d indico

db-reset: ## Reset database
	@echo "Resetting database..."
	@$(DOCKER_COMPOSE) down postgres
	@docker volume rm backend_postgres_data 2>/dev/null || true
	@$(DOCKER_COMPOSE) up -d postgres
	@echo "Database reset complete!"

clean: ## Clean up development environment
	@echo "Cleaning up..."
	@$(DOCKER_COMPOSE) down -v
	@docker system prune -f
	@$(GO) clean -cache
	@rm -f bin/server bin/seeder coverage.out coverage.html
	@echo "Cleanup complete!"

# API testing targets
api-health: ## Test health endpoint
	@curl -s http://localhost:8080/health | jq .

api-order: ## Create test order
	@curl -s -X POST http://localhost:8080/orders \
		-H "Content-Type: application/json" \
		-d '{"product_id":1,"quantity":1,"buyer_id":"test_buyer"}' | jq .

api-settlement: ## Create settlement job
	@curl -s -X POST http://localhost:8080/jobs/settlement \
		-H "Content-Type: application/json" \
		-d '{"from":"2025-01-01","to":"2025-01-31"}' | jq .

# Install development tools
install-tools: ## Install development tools
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Tools installed!"

# Performance testing
perf-test: ## Run performance test with concurrent orders
	@echo "Running performance test (500 concurrent orders)..."
	@for i in $$(seq 1 500); do \
		curl -s -X POST http://localhost:8080/orders \
			-H "Content-Type: application/json" \
			-d "{\"product_id\":1,\"quantity\":1,\"buyer_id\":\"buyer_$$i\"}" > /dev/null & \
	done; \
	wait
	@echo "Performance test complete. Check logs for results."

# Documentation
docs: ## Generate documentation
	@echo "Generating documentation..."
	@godoc -http=:6060 &
	@echo "Documentation server started at http://localhost:6060"
