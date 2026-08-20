-include .env
export

.PHONY: run build test test-integration test-all lint check clean \
       docker-up docker-down migrate-up migrate-down migrate-create \
       migrate-go-up migrate-go-down build-migrate \
       coverage fmt vet

APP_NAME := task-tracker-api
BUILD_DIR := ./bin

run:
	go run ./cmd/api

build:
	go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/api

test:
	go test ./...

test-integration:
	go test -tags integration -p 4 ./...

test-all:
	go test -tags integration ./...

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed, skipping lint"; \
	fi

check: vet lint test

clean:
	rm -rf $(BUILD_DIR)

# --- Docker ---

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down

# --- Migrations (requires golang-migrate CLI) ---

migrate-up:
	migrate -source file://migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -source file://migrations -database "$(DATABASE_URL)" down 1

# Usage: make migrate-create name=create_users
migrate-create:
	migrate create -ext sql -dir migrations -seq $(name)

# --- Migrations (Go binary, no external CLI needed) ---

migrate-go-up:
	go run ./cmd/migrate -direction=up

migrate-go-down:
	go run ./cmd/migrate -direction=down -steps=1

build-migrate:
	go build -o $(BUILD_DIR)/migrate ./cmd/migrate

# --- Code quality ---

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

fmt:
	go fmt ./...

vet:
	go vet ./...
