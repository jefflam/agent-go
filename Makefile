.PHONY: run build clean migrate-create migrate-up migrate-down migrate-status

# Load environment variables from .env file
ifneq (,$(wildcard .env))
    include .env
    export
endif

# Default binary name
BINARY_NAME=agent
BUILD_DIR=build

# Build the application
build:
	@echo "Building..."
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) cmd/agent/main.go

# Run the application
run: clean build
	@echo "Running..."
	@./$(BUILD_DIR)/$(BINARY_NAME)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)

# Run with direct go run (faster for development)
dev:
	@echo "Running in dev mode..."
	@go run cmd/agent/main.go

# Database connection string
DB_URL="postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable"

# Check required environment variables
check-db-env:
	@if [ -z "$(DB_USER)" ]; then echo "DB_USER is not set"; exit 1; fi
	@if [ -z "$(DB_PASSWORD)" ]; then echo "DB_PASSWORD is not set"; exit 1; fi
	@if [ -z "$(DB_HOST)" ]; then echo "DB_HOST is not set"; exit 1; fi
	@if [ -z "$(DB_PORT)" ]; then echo "DB_PORT is not set"; exit 1; fi
	@if [ -z "$(DB_NAME)" ]; then echo "DB_NAME is not set"; exit 1; fi

migrate-create:
	@read -p "Enter migration name: " name; \
	migrate create -ext sql -dir migrations -seq $$name

migrate-up: check-db-env
	migrate -path migrations -database "$(DB_URL)" up

migrate-down: check-db-env
	migrate -path migrations -database "$(DB_URL)" down

migrate-status: check-db-env
	migrate -path migrations -database "$(DB_URL)" version