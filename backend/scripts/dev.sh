#!/bin/bash

# Indico Backend Development Scripts

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Docker is running
check_docker() {
    if ! docker info > /dev/null 2>&1; then
        log_error "Docker is not running. Please start Docker first."
        exit 1
    fi
}

# Development commands
dev_setup() {
    log_info "Setting up development environment..."
    check_docker
    
    # Copy environment file if it doesn't exist
    if [ ! -f .env ]; then
        cp .env.example .env
        log_info "Created .env file from .env.example"
    fi
    
    # Start services
    docker-compose up -d postgres postgres_test
    log_info "Started PostgreSQL services"
    
    # Wait for PostgreSQL to be ready
    log_info "Waiting for PostgreSQL to be ready..."
    sleep 5
    
    # Install Go dependencies
    go mod download
    log_info "Downloaded Go dependencies"
    
    log_info "Development environment setup complete!"
    log_info "You can now run: make dev"
}

dev_start() {
    log_info "Starting development server..."
    check_docker
    
    # Start PostgreSQL if not running
    docker-compose up -d postgres
    
    # Run the application
    go run cmd/server/main.go
}

dev_test() {
    log_info "Running tests..."
    check_docker
    
    # Start test database
    docker-compose up -d postgres_test
    
    # Wait for database to be ready
    sleep 3
    
    # Run tests
    go test ./test/... -v
}

dev_seed() {
    log_info "Seeding test data..."
    check_docker
    
    # Start PostgreSQL if not running
    docker-compose up -d postgres
    
    # Wait for database to be ready
    sleep 3
    
    # Run seeder
    go run cmd/seeder/main.go
}

dev_clean() {
    log_info "Cleaning up development environment..."
    
    # Stop and remove containers
    docker-compose down
    
    # Remove volumes
    docker-compose down -v
    
    # Clean Go cache
    go clean -cache
    
    log_info "Development environment cleaned"
}

prod_build() {
    log_info "Building production image..."
    check_docker
    
    docker build -t indico-backend .
    log_info "Production image built successfully"
}

prod_start() {
    log_info "Starting production services..."
    check_docker
    
    docker-compose up -d
    log_info "Production services started"
    log_info "API available at: http://localhost:8080"
    log_info "Grafana available at: http://localhost:3000 (admin/admin)"
    log_info "Prometheus available at: http://localhost:9090"
}

prod_logs() {
    docker-compose logs -f app
}

prod_stop() {
    log_info "Stopping production services..."
    docker-compose down
}

# API testing helpers
test_health() {
    log_info "Testing health endpoint..."
    curl -s http://localhost:8080/health | jq .
}

test_order() {
    log_info "Creating test order..."
    curl -s -X POST http://localhost:8080/orders \
        -H "Content-Type: application/json" \
        -d '{"product_id":1,"quantity":1,"buyer_id":"test_buyer"}' | jq .
}

test_concurrent_orders() {
    log_info "Testing concurrent orders (500 requests)..."
    
    for i in {1..500}; do
        curl -s -X POST http://localhost:8080/orders \
            -H "Content-Type: application/json" \
            -d "{\"product_id\":1,\"quantity\":1,\"buyer_id\":\"buyer_$i\"}" > /dev/null &
    done
    
    wait
    log_info "All 500 requests sent. Check logs for results."
}

test_settlement_job() {
    log_info "Creating settlement job..."
    job_response=$(curl -s -X POST http://localhost:8080/jobs/settlement \
        -H "Content-Type: application/json" \
        -d '{"from":"2025-01-01","to":"2025-01-31"}')
    
    job_id=$(echo $job_response | jq -r .job_id)
    log_info "Created job: $job_id"
    
    # Poll for completion
    log_info "Polling job status..."
    while true; do
        status=$(curl -s "http://localhost:8080/jobs/$job_id" | jq -r .status)
        echo "Job status: $status"
        
        if [ "$status" = "COMPLETED" ] || [ "$status" = "FAILED" ]; then
            break
        fi
        
        sleep 2
    done
    
    # Show final result
    curl -s "http://localhost:8080/jobs/$job_id" | jq .
}

# Database helpers
db_migrate() {
    log_info "Running database migrations..."
    check_docker
    
    # Ensure PostgreSQL is running
    docker-compose up -d postgres
    
    # Apply migrations manually (since we're not using a migration tool)
    docker-compose exec postgres psql -U postgres -d indico -f /docker-entrypoint-initdb.d/001_initial_schema.up.sql
}

db_reset() {
    log_info "Resetting database..."
    check_docker
    
    docker-compose down postgres
    docker volume rm backend_postgres_data 2>/dev/null || true
    docker-compose up -d postgres
    
    log_info "Database reset complete"
}

db_shell() {
    log_info "Opening database shell..."
    docker-compose exec postgres psql -U postgres -d indico
}

# Show help
show_help() {
    echo "Indico Backend Development Scripts"
    echo ""
    echo "Development Commands:"
    echo "  setup      - Set up development environment"
    echo "  start      - Start development server"
    echo "  test       - Run tests"
    echo "  seed       - Seed test data"
    echo "  clean      - Clean development environment"
    echo ""
    echo "Production Commands:"
    echo "  build      - Build production Docker image"
    echo "  prod       - Start production services"
    echo "  logs       - Show production logs"
    echo "  stop       - Stop production services"
    echo ""
    echo "Testing Commands:"
    echo "  test-health     - Test health endpoint"
    echo "  test-order      - Create a test order"
    echo "  test-concurrent - Test 500 concurrent orders"
    echo "  test-settlement - Test settlement job processing"
    echo ""
    echo "Database Commands:"
    echo "  db-migrate - Run database migrations"
    echo "  db-reset   - Reset database"
    echo "  db-shell   - Open database shell"
    echo ""
    echo "Usage: ./scripts/dev.sh <command>"
}

# Main script logic
case "$1" in
    setup)
        dev_setup
        ;;
    start)
        dev_start
        ;;
    test)
        dev_test
        ;;
    seed)
        dev_seed
        ;;
    clean)
        dev_clean
        ;;
    build)
        prod_build
        ;;
    prod)
        prod_start
        ;;
    logs)
        prod_logs
        ;;
    stop)
        prod_stop
        ;;
    test-health)
        test_health
        ;;
    test-order)
        test_order
        ;;
    test-concurrent)
        test_concurrent_orders
        ;;
    test-settlement)
        test_settlement_job
        ;;
    db-migrate)
        db_migrate
        ;;
    db-reset)
        db_reset
        ;;
    db-shell)
        db_shell
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        log_error "Unknown command: $1"
        show_help
        exit 1
        ;;
esac
