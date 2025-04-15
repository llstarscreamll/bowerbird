.PHONY: build-api deploy all help

# Default target when just running 'make'
all: build-lambda deploy

# Help message
help:
	@echo "Available commands:"
	@echo "  make build-lambda  - Build Lambda function"
	@echo "  make deploy        - Deploy to AWS using CDK"

# Build API Server
build-api:
	@echo "Building Lambda function..."
	@docker compose up --remove-orphans --quiet-pull build-api

# Build SPA
build-spa:
	@echo "Building SPA..."
	@docker compose up --quiet-pull build-spa

# Deploy
deploy: build-api build-spa
	@echo "Deploying project..."
	@docker compose up --remove-orphans --quiet-pull deploy
