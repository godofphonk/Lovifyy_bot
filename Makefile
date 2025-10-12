# Lovifyy Bot Makefile
# Professional development workflow

.PHONY: help build test clean run docker-build docker-run lint fmt vet deps security coverage

# Variables
BINARY_NAME=lovifyy_bot
DOCKER_IMAGE=lovifyy-bot
VERSION=$(shell git describe --tags --always --dirty)
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse HEAD)
LDFLAGS=-ldflags "-X main.version=${VERSION} -X main.buildTime=${BUILD_TIME} -X main.gitCommit=${GIT_COMMIT}"

# Default target
help: ## Show this help message
	@echo "Lovifyy Bot - Professional Development Workflow"
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Development
build: ## Build the application
	@echo "🔨 Building Lovifyy Bot..."
	@mkdir -p build
	go build ${LDFLAGS} -o build/${BINARY_NAME} cmd/main.go

run: ## Run the application
	@echo "🚀 Running Lovifyy Bot..."
	go run ${LDFLAGS} cmd/main.go

# Testing
test: ## Run all tests
	@echo "🧪 Running tests..."
	go test -v -race -coverprofile=coverage.out ./...

test-integration: ## Run integration tests
	@echo "🔗 Running integration tests..."
	go test -v -tags=integration ./tests/

coverage: test ## Generate coverage report
	@echo "📊 Generating coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

benchmark: ## Run benchmarks
	@echo "⚡ Running benchmarks..."
	go test -bench=. -benchmem ./...

# Code Quality
lint: ## Run linter
	@echo "🔍 Running linter..."
	golangci-lint run --timeout=5m

fmt: ## Format code
	@echo "✨ Formatting code..."
	go fmt ./...
	goimports -w .

vet: ## Run go vet
	@echo "🔎 Running go vet..."
	go vet ./...

security: ## Run security scan
	@echo "🔒 Running security scan..."
	gosec ./...

# Dependencies
deps: ## Download dependencies
	@echo "📦 Downloading dependencies..."
	go mod download
	go mod tidy

deps-update: ## Update dependencies
	@echo "🔄 Updating dependencies..."
	go get -u ./...
	go mod tidy

# Docker
docker-build: ## Build Docker image
	@echo "🐳 Building Docker image..."
	docker build -t ${DOCKER_IMAGE}:${VERSION} -t ${DOCKER_IMAGE}:latest .

docker-run: ## Run Docker container
	@echo "🐳 Running Docker container..."
	docker run --rm --env-file .env -v $(PWD)/data:/app/data -v $(PWD)/exercises:/app/exercises ${DOCKER_IMAGE}:latest

docker-compose-up: ## Start with docker-compose
	@echo "🐳 Starting with docker-compose..."
	docker-compose up -d

docker-compose-down: ## Stop docker-compose
	@echo "🐳 Stopping docker-compose..."
	docker-compose down

docker-compose-logs: ## Show docker-compose logs
	@echo "📋 Showing docker-compose logs..."
	docker-compose logs -f

# Production
deploy: ## Deploy to production
	@echo "🚀 Deploying to production..."
	./deployment/deploy.sh

deploy-update: ## Update production deployment
	@echo "🔄 Updating production deployment..."
	./deployment/update.sh

# Utilities
clean: ## Clean build artifacts
	@echo "🧹 Cleaning build artifacts..."
	rm -rf build/ bin/
	rm -f coverage.out coverage.html
	go clean -cache -testcache -modcache

install-tools: ## Install development tools
	@echo "🛠️ Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest

setup: deps install-tools ## Setup development environment
	@echo "⚙️ Setting up development environment..."
	cp examples/config.example.json config.json
	cp examples/.env.example .env
	mkdir -p data/logs data/chats data/diaries data/backups data/notifications exercises
	@echo "✅ Development environment setup complete!"
	@echo "📝 Don't forget to edit .env and config.json with your tokens!"

# Database
backup: ## Create backup
	@echo "💾 Creating backup..."
	mkdir -p data/backups
	tar -czf data/backups/backup-$(shell date +%Y%m%d-%H%M%S).tar.gz data/ exercises/

restore: ## Restore from backup (specify BACKUP_FILE)
	@echo "📥 Restoring from backup..."
	@if [ -z "$(BACKUP_FILE)" ]; then echo "❌ Please specify BACKUP_FILE=path/to/backup.tar.gz"; exit 1; fi
	tar -xzf $(BACKUP_FILE)

# Monitoring
health: ## Check application health
	@echo "🏥 Checking application health..."
	curl -f http://localhost:8081/health || echo "❌ Health check failed"

metrics: ## Show metrics
	@echo "📊 Showing metrics..."
	curl -s http://localhost:9090/metrics | head -20

logs: ## Show application logs
	@echo "📋 Showing application logs..."
	tail -f data/logs/lovifyy_bot.log

# Release
release: test lint security ## Prepare release
	@echo "🎉 Preparing release..."
	@if [ -z "$(TAG)" ]; then echo "❌ Please specify TAG=v1.0.0"; exit 1; fi
	git tag -a $(TAG) -m "Release $(TAG)"
	git push origin $(TAG)
	@echo "✅ Release $(TAG) created!"

# Development workflow
dev: fmt vet test ## Run development workflow
	@echo "✅ Development workflow completed!"

ci: deps fmt vet lint test security coverage ## Run CI workflow
	@echo "✅ CI workflow completed!"


# Help with colors
.DEFAULT_GOAL := help
