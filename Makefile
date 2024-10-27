.PHONY: run build clean

# Default binary name
BINARY_NAME=agent
BUILD_DIR=build

# Build the application
build:
	@echo "Building..."
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) cmd/agent/main.go

# Run the application
run: build
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