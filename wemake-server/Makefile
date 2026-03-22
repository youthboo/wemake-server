.PHONY: help build run test clean docker-up docker-down

help:
	@echo "Wemake - Factory & Entrepreneur Connection Platform"
	@echo ""
	@echo "Available commands:"
	@echo "  make build       - Build the application"
	@echo "  make run         - Run the application"
	@echo "  make test        - Run tests"
	@echo "  make clean       - Clean build artifacts"
	@echo "  make docker-up   - Start Docker containers"
	@echo "  make docker-down - Stop Docker containers"
	@echo "  make deps        - Download dependencies"

deps:
	go mod download
	go mod tidy

build:
	go build -o bin/wemake ./cmd/app

run:
	go run ./cmd/app/main.go

test:
	go test -v ./...

clean:
	rm -rf bin/
	go clean

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

dev:
	go run ./cmd/app/main.go
