.PHONY: help api serve test build clean docker docker-run docker-compose-up docker-compose-down coverage bench trivy trivy-fs stress-test stress-test-light stress-test-heavy cache-stats cache-clear

# Default target
.DEFAULT_GOAL := help

# Binary name
BINARY_NAME=packcalc
BUILD_DIR=bin

# Docker Compose compatibility: detect if docker-compose (v1) or docker compose (v2) is available
DOCKER_COMPOSE := $(shell command -v docker-compose 2> /dev/null)
ifdef DOCKER_COMPOSE
    DC = docker-compose
else
    DC = docker compose
endif

help: ## Show this help message
	@echo "Pack Calculator - Available commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

api: ## Run API server only (port 8080)
	@echo "🚀 Starting API server..."
	go run main.go api

serve: ## Run web UI + API server (port 8080)
	@echo "🚀 Starting web UI + API server..."
	go run main.go serve

test: ## Run all tests
	@echo "🧪 Running tests..."
	go test -v ./...

coverage: ## Run tests with coverage
	@echo "📊 Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

bench: ## Run benchmark tests
	@echo "⚡ Running benchmarks..."
	go test -bench=. -benchmem ./internal/algorithm/

build: ## Build binary
	@echo "🔨 Building binary..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) main.go
	@echo "✅ Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

build-linux: ## Build binary for Linux
	@echo "🔨 Building binary for Linux..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux main.go
	@echo "✅ Binary built: $(BUILD_DIR)/$(BINARY_NAME)-linux"

clean: ## Clean build artifacts and data
	@echo "🧹 Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -rf data/*.db
	rm -f coverage.out coverage.html
	@echo "✅ Cleaned"

docker: ## Build Docker image
	@echo "🐳 Building Docker image..."
	docker build -t packcalc:latest .
	@echo "✅ Docker image built: packcalc:latest"

docker-run: docker ## Build and run Docker container
	@echo "🐳 Running Docker container..."
	docker run -p 8080:8080 --name packcalc-container packcalc:latest

docker-stop: ## Stop and remove Docker container
	@echo "🛑 Stopping Docker container..."
	docker stop packcalc-container || true
	docker rm packcalc-container || true

docker-clean: docker-stop ## Clean Docker images and containers
	@echo "🧹 Cleaning Docker..."
	docker rmi packcalc:latest || true

# Security scanning
# Trivy can run locally (if installed) or via Docker container
TRIVY_LOCAL := $(shell command -v trivy 2> /dev/null)
ifdef TRIVY_LOCAL
    TRIVY = trivy
else
    TRIVY = docker run --rm -v /var/run/docker.sock:/var/run/docker.sock -v $(PWD):/workspace aquasec/trivy:latest
endif

trivy: ## Run Trivy vulnerability scan on Docker image
	@echo "🔍 Running Trivy vulnerability scan..."
	@if [ -z "$(TRIVY_LOCAL)" ]; then \
		echo "ℹ️  Using Trivy Docker container (trivy not installed locally)"; \
	fi
	$(TRIVY) image --severity HIGH,CRITICAL packcalc:latest

trivy-fs: ## Run Trivy filesystem scan on source code
	@echo "🔍 Running Trivy filesystem scan..."
	@if [ -z "$(TRIVY_LOCAL)" ]; then \
		echo "ℹ️  Using Trivy Docker container (trivy not installed locally)"; \
		$(TRIVY) fs --severity HIGH,CRITICAL /workspace; \
	else \
		$(TRIVY) fs --severity HIGH,CRITICAL .; \
	fi

trivy-config: ## Run Trivy config scan (Dockerfile, docker-compose.yml)
	@echo "🔍 Running Trivy config scan..."
	@if [ -z "$(TRIVY_LOCAL)" ]; then \
		echo "ℹ️  Using Trivy Docker container (trivy not installed locally)"; \
		$(TRIVY) config /workspace; \
	else \
		$(TRIVY) config .; \
	fi

trivy-all: docker trivy trivy-fs trivy-config ## Run all Trivy scans (image, filesystem, config)
	@echo "✅ All Trivy scans completed"

# Docker Compose commands (auto-detects v1 or v2)
up: ## Start all services with docker compose (Web UI + API)
	@echo "🚀 Starting services with $(DC)..."
	$(DC) up -d
	@echo "✅ Services started!"
	@echo "🌐 Web UI: http://localhost:8080"
	@echo "📊 API: http://localhost:8080/api"
	@echo "💚 Health: http://localhost:8080/api/health"

down: ## Stop all services with docker compose
	@echo "🛑 Stopping services..."
	$(DC) down
	@echo "✅ Services stopped"

logs: ## Show logs from docker compose services
	@echo "📋 Showing logs..."
	$(DC) logs -f

ps: ## Show running docker compose services
	@echo "📊 Running services:"
	$(DC) ps

restart: ## Restart all services
	@echo "🔄 Restarting services..."
	$(DC) restart
	@echo "✅ Services restarted"

up-api: ## Start API-only service with docker compose
	@echo "🚀 Starting API-only service..."
	$(DC) --profile api-only up -d api
	@echo "✅ API service started on http://localhost:8081"

rebuild: ## Rebuild and restart services
	@echo "🔨 Rebuilding services..."
	$(DC) up -d --build
	@echo "✅ Services rebuilt and restarted"

install: build ## Install binary to $GOPATH/bin
	@echo "📦 Installing binary..."
	go install
	@echo "✅ Installed to $(shell go env GOPATH)/bin/$(BINARY_NAME)"

deps: ## Download dependencies
	@echo "📦 Downloading dependencies..."
	go mod download
	go mod tidy
	@echo "✅ Dependencies updated"

fmt: ## Format code
	@echo "🎨 Formatting code..."
	go fmt ./...
	@echo "✅ Code formatted"

lint: ## Run linter (requires golangci-lint)
	@echo "🔍 Running linter..."
	golangci-lint run || echo "⚠️  golangci-lint not installed. Run: brew install golangci-lint"

validate-edge-case: build ## Validate the edge case requirement
	@echo "🧪 Validating edge case: 500,000 items with pack sizes [23, 31, 53]"
	@echo "Expected: {23: 2, 31: 7, 53: 9429}"
	@$(BUILD_DIR)/$(BINARY_NAME) serve &
	@sleep 2
	@curl -s -X POST http://localhost:8080/api/packs/config -H "Content-Type: application/json" -d '{"pack_sizes":[23,31,53]}' > /dev/null
	@curl -s -X POST http://localhost:8080/api/calculate -H "Content-Type: application/json" -d '{"items":500000}' | jq '.result'
	@pkill -f "$(BINARY_NAME) serve" || true

dev: ## Run in development mode with auto-reload (requires air)
	@echo "🔄 Starting development server..."
	air || echo "⚠️  air not installed. Run: go install github.com/cosmtrek/air@latest"

all: clean deps test build ## Clean, download deps, test, and build

.PHONY: all

# ============================================================================
# Stress Testing & Performance
# ============================================================================

stress-test: ## Run stress test (1000 requests, 50 concurrent) - uses Docker, no local tools needed
	@echo "🔥 Running stress test..."
	@echo ""
	@echo "📊 Testing /api/calculate endpoint..."
	@echo "   - 1000 requests"
	@echo "   - 50 concurrent workers"
	@echo "   - Testing cache performance"
	@echo "   - Using Docker (no local tools required)"
	@echo ""
	@docker run --rm --network=host williamyeh/hey:latest \
		-n 1000 -c 50 -m POST -H "Content-Type: application/json" \
		-d '{"items": 251}' \
		http://localhost:8080/api/calculate
	@echo ""
	@echo "✅ Stress test complete! Check cache stats with: make cache-stats"

stress-test-light: ## Run light stress test (100 requests, 10 concurrent) - uses Docker
	@echo "🔥 Running light stress test..."
	@docker run --rm --network=host williamyeh/hey:latest \
		-n 100 -c 10 -m POST -H "Content-Type: application/json" \
		-d '{"items": 251}' \
		http://localhost:8080/api/calculate

stress-test-heavy: ## Run heavy stress test (10000 requests, 100 concurrent) - uses Docker
	@echo "🔥 Running HEAVY stress test..."
	@echo ""
	@echo "⚠️  WARNING: This will send 10,000 requests with 100 concurrent workers"
	@echo "   Press Ctrl+C to cancel, or wait 3 seconds to continue..."
	@sleep 3
	@echo ""
	@docker run --rm --network=host williamyeh/hey:latest \
		-n 10000 -c 100 -m POST -H "Content-Type: application/json" \
		-d '{"items": 251}' \
		http://localhost:8080/api/calculate
	@echo ""
	@echo "✅ Heavy stress test complete! Check cache stats with: make cache-stats"

cache-stats: ## Show cache statistics - uses Docker
	@echo "📊 Cache Statistics:"
	@echo ""
	@docker run --rm --network=host curlimages/curl:latest \
		-s http://localhost:8080/api/cache/stats | \
		docker run --rm -i ghcr.io/jqlang/jq:latest '.' || \
		echo "❌ Failed to get cache stats. Is the server running?"

cache-clear: ## Clear all cache - uses Docker
	@echo "🗑️  Clearing cache..."
	@docker run --rm --network=host curlimages/curl:latest \
		-s -X POST http://localhost:8080/api/cache/clear | \
		docker run --rm -i ghcr.io/jqlang/jq:latest '.' || \
		echo "❌ Failed to clear cache. Is the server running?"

