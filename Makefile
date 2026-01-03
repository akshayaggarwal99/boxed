# Boxed - Development Workflow

.PHONY: all build build-agent build-cli test clean

# Default: Build everything
all: build

# Build the unified 'boxed' CLI
build: build-agent
	@echo "Building Control Plane (Go)..."
	go build -o bin/boxed ./cmd/boxed

# Build the Rust Agent (Linux aarch64 inside Docker)
# This ensures it's compatible with the default Docker driver
build-agent:
	@echo "Building Boxed Agent (Rust) for Linux..."
	mkdir -p bin
	docker run --rm -v "$(shell pwd)":/app -w /app/agent rust:1.80-slim cargo build --release
	cp agent/target/release/boxed-agent bin/boxed-agent
	@echo "Agent binary placed in bin/boxed-agent"

# Run all integration tests
test: build
	@echo "Running Integration Tests..."
	lsof -i :8081 -t | xargs kill -9 || true
	go test -v ./tests/integration/...

# Clean build artifacts
clean:
	@echo "Cleaning artifacts..."
	rm -rf bin/
	rm -rf agent/target
	rm -rf sdk/typescript/dist sdk/typescript/node_modules
	rm -rf sdk/python/dist sdk/python/build sdk/python/*.egg-info
	go clean
	@echo "Clean complete."
