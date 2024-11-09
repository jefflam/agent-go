.PHONY: run build clean migrate-create migrate-up migrate-down

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


migrate-create:
	@read -p "Enter migration name: " name; \
	migrate create -ext sql -dir migrations -seq $$name

migrate-up:
	migrate -path migrations -database "postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable" up

migrate-down:
	migrate -path migrations -database "postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable" down