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
	@docker compose up --build --remove-orphans --quiet-pull build-api

# Deploy
deploy: build-api
	@echo "Deploying project..."
	@docker compose up --build --remove-orphans --quiet-pull deploy
