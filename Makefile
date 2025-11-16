.PHONY: help dev build up down logs clean restart shell test

# Default target
help: ## Show this help message
	@echo "Workshop Inventory - Development Commands"
	@echo "========================================"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

dev: ## Start development environment
	@echo "🚀 Starting Workshop Inventory development environment..."
	docker-compose up --build -d
	@echo "✅ Application running at: http://localhost:9090"
	@echo "📁 File browser (debug): http://localhost:9091 (use 'make debug' to enable)"

debug: ## Start with debug tools (file browser)
	@echo "🔧 Starting development environment with debug tools..."
	docker-compose --profile debug up --build -d
	@echo "✅ Application running at: http://localhost:9090"
	@echo "📁 File browser available at: http://localhost:9091"

build: ## Build the application
	docker-compose build

up: ## Start services (without rebuild)
	docker-compose up -d

down: ## Stop all services
	@echo "🛑 Stopping services..."
	docker-compose down

logs: ## Show application logs
	docker-compose logs -f workshop-inventory

logs-all: ## Show all services logs
	docker-compose logs -f

clean: ## Clean up containers, volumes, and images
	@echo "🧹 Cleaning up..."
	docker-compose down -v --remove-orphans
	docker system prune -f

restart: ## Restart the application
	@echo "🔄 Restarting application..."
	docker-compose restart workshop-inventory

shell: ## Access application container shell
	docker-compose exec workshop-inventory sh

test: ## Run tests (if available)
	@echo "🧪 Running tests..."
	go test ./...

# Development helpers
init-data: ## Initialize with sample data (creates empty data files if they don't exist)
	@echo "📝 Initializing data files..."
	@if [ ! -f dados.json ]; then echo "[]" > dados.json; echo "Created dados.json"; fi
	@if [ ! -f usuarios.json ]; then echo "[]" > usuarios.json; echo "Created usuarios.json"; fi
	@mkdir -p static/photos/thumbs
	@echo "✅ Data files initialized"

status: ## Show container status
	docker-compose ps

setup: init-data dev ## Complete setup: initialize data and start development environment