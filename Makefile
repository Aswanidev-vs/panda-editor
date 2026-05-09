# Panda Editor Makefile

BINARY_NAME=panda.exe
VERSION=1.0.0
BUILD_DIR=build

.PHONY: all build clean run test

all: build

build:
	@echo "Building Panda Editor v$(VERSION)..."
	@go build -o $(BINARY_NAME) main.go
	@echo "Build complete: $(BINARY_NAME)"

release:
	@echo "Building release binary..."
	@mkdir -p $(BUILD_DIR)
	@go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) main.go
	@echo "Release binary created in $(BUILD_DIR)/"

clean:
	@if exist $(BINARY_NAME) del $(BINARY_NAME)
	@if exist $(BUILD_DIR) rmdir /s /q $(BUILD_DIR)
	@echo "Cleaned up build artifacts."

run: build
	@./$(BINARY_NAME)

test:
	@go test ./...
