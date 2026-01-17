.PHONY: help build run run-local dev test clean proto config-validate

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the server binary
	@echo "Building server..."
	@go build -o bin/server ./backend/cmd/main.go
	@echo "✓ Build complete: bin/server"

run: ## Run the server (production mode)
	@echo "Starting server..."
	@go run ./backend/cmd/main.go

run-local: ## Run the server in local development mode
	@echo "Starting server in LOCAL_MODE..."
	@LOCAL_MODE=true go run ./backend/cmd/main.go

dev: config-validate run-local ## Validate config and run in development mode

test: ## Run tests
	@echo "Running tests..."
	@go test -v ./...

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -rf backend/data/*.db
	@echo "✓ Clean complete"

proto: ## Generate protobuf code
	@echo "Generating protobuf code..."
	@cd backend && buf generate
	@echo "✓ Protobuf generation complete"

config-validate: ## Validate configuration
	@echo "Validating configuration..."
	@go run ./backend/cmd/config-validator/main.go

config-init: ## Initialize config from example
	@if [ ! -f backend/config/config.yaml ]; then \
		echo "Creating backend/config/config.yaml from example..."; \
		cp backend/config/config.example.yaml backend/config/config.yaml; \
		echo "✓ backend/config/config.yaml created"; \
		echo "⚠️  Please edit backend/config/config.yaml to match your environment"; \
	else \
		echo "backend/config/config.yaml already exists"; \
	fi

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@GOSUMDB=sum.golang.org go mod download
	@echo "✓ Dependencies downloaded"

tidy: ## Tidy go.mod
	@echo "Tidying go.mod..."
	@GOSUMDB=sum.golang.org go mod tidy
	@echo "✓ go.mod tidied"

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t algorithm-platform:latest -f deploy/Dockerfile .
	@echo "✓ Docker image built"

docker-up: ## Start services with docker-compose
	@echo "Starting services..."
	@docker-compose -f deploy/docker-compose.yml up -d
	@echo "✓ Services started"

docker-down: ## Stop services with docker-compose
	@echo "Stopping services..."
	@docker-compose -f deploy/docker-compose.yml down
	@echo "✓ Services stopped"

docker-logs: ## Show docker-compose logs
	@docker-compose -f deploy/docker-compose.yml logs -f

install: deps config-init ## Install dependencies and initialize config
	@echo "✓ Installation complete"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Edit backend/config/config.yaml to match your environment"
	@echo "  2. Run 'make run-local' to start the server"
