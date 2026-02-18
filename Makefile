.PHONY: dev build test migrate-up migrate-down sqlc lint clean

# Config
APP_NAME := terrascore-api
MAIN := ./cmd/server
VERSION := $(shell cat VERSION 2>/dev/null || echo dev)
MIGRATE := ./cmd/migrate
DB_URL ?= postgres://terrascore:terrascore@localhost:5432/terrascore?sslmode=disable

# Development — start infra + Go server
dev:
	docker compose up -d postgres redis keycloak kong
	@echo "Waiting for services to be healthy..."
	@sleep 5
	go run $(MAIN)

# Build Go binary
build:
	CGO_ENABLED=0 go build -ldflags="-X main.Version=$(VERSION)" -o bin/$(APP_NAME) $(MAIN)

# Run tests
test:
	go test ./... -v -race -count=1

# Run tests with coverage
test-cover:
	go test ./... -race -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

# Database migrations
migrate-up:
	go run $(MIGRATE) -direction up -db "$(DB_URL)"

migrate-down:
	go run $(MIGRATE) -direction down -db "$(DB_URL)" -steps 1

migrate-create:
	@read -p "Migration name: " name; \
	touch db/migrations/$$(printf "%03d" $$(($$(ls db/migrations/*.up.sql 2>/dev/null | wc -l) + 1)))_$${name}.up.sql; \
	touch db/migrations/$$(printf "%03d" $$(($$(ls db/migrations/*.up.sql 2>/dev/null | wc -l))))_$${name}.down.sql

# sqlc code generation
sqlc:
	sqlc generate

# Linting
lint:
	go vet ./...
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run || echo "golangci-lint not installed, skipping"

# Clean build artifacts
clean:
	rm -rf bin/ coverage.out coverage.html

# Docker — full stack
up:
	docker compose up -d

down:
	docker compose down

# Docker — rebuild Go app
docker-build:
	docker build -t $(APP_NAME):latest .
