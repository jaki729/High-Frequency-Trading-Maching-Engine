.PHONY: build run test clean docker-build docker-run

# Build the application
build:
	go build -o bin/hft-engine cmd/main.go

# Run the application
run: build
	./bin/hft-engine

# Run tests
test:
	go test -v ./...

# Run benchmarks
bench:
	go test -bench=. ./engine/

# Clean build artifacts
clean:
	rm -rf bin/

# Build Docker image
docker-build:
	docker build -t hft-matching-engine .

# Run with Docker Compose
docker-run:
	docker-compose up -d

# Stop Docker services
docker-stop:
	docker-compose down

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Generate mocks (if using gomock)
generate:
	go generate ./...

# Install dependencies
deps:
	go mod tidy
	go mod download