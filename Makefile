.PHONY: check
check: clean build fmt vet unit
	@echo "Check completed successfully!"

.PHONY: clean
clean:
	@echo "Cleaning..."
	@go clean
	@rm -f manto-web

.PHONY: build
build:
	@echo "Building..."
	@go build -o manto-web ./cmd/manto-web

.PHONY: vet
vet:
	@echo "Running go vet..."
	@go vet ./...

.PHONY: fmt
fmt:
	@echo "Formatting code..."
	@go fmt ./...

.PHONY: unit
unit:
	@echo "Running unit tests..."
	@go test ./internal/... -v

.PHONY: start
start: build
	@echo "Starting application in production mode..."
	@GO_ENV=production ./manto-web

.PHONY: start-dev
start-dev: build
	@echo "Starting application in development mode..."
	@GO_ENV=development ./manto-web

.PHONY: integration
integration:
	@echo "Running integration tests..."
	@go test . -v

.PHONY: test
test:
	@echo "Running all tests..."
	@go test ./... -v

.PHONY: help
help:
	@echo "Available commands:"
	@echo "  check           - Clean, build, fmt, vet and run unit tests"
	@echo "  clean           - Clean build artifacts"
	@echo "  build           - Build the application"
	@echo "  vet             - Run go vet"
	@echo "  fmt             - Format code"
	@echo "  unit            - Run unit tests only"
	@echo "  start           - Start application in production mode"
	@echo "  start-dev       - Start application in development mode"
	@echo "  integration     - Run integration tests only"
	@echo "  test            - Run all tests"
	@echo "  help            - Show this help message"
