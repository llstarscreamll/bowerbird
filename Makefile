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
deploy: build-spa
	@echo "Deploying project $(APP_DOMAIN_NAME)..."
	@DOCKER_BUILDKIT=1 BUILDX_NO_DEFAULT_ATTESTATIONS=1 docker compose up --remove-orphans --quiet-pull deploy
