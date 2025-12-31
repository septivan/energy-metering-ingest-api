.PHONY: help build run test clean docker-build docker-run

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the application
	go build -o bin/server cmd/server/main.go

run: ## Run the application locally
	go run cmd/server/main.go

test: ## Run tests
	go test -v -race -coverprofile=coverage.out ./...

test-cover: test ## Run tests with coverage report
	go tool cover -html=coverage.out

clean: ## Clean build artifacts
	rm -rf bin/ coverage.out

tidy: ## Tidy go modules
	go mod tidy

fmt: ## Format code
	go fmt ./...

lint: ## Run linter (requires golangci-lint)
	golangci-lint run

docker-build: ## Build Docker image
	docker build -t energy-metering-ingest-api:latest .

docker-run: ## Run Docker container locally
	docker run -p 8080:8080 \
		-e RABBITMQ_URL="${RABBITMQ_URL}" \
		energy-metering-ingest-api:latest

fly-deploy: ## Deploy to Fly.io
	fly deploy

fly-logs: ## View Fly.io logs
	fly logs

fly-status: ## Check Fly.io status
	fly status
