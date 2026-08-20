# beer-lms-api Makefile — copy to the Go module root (beer-lms-api/Makefile).
# This is the real target set the repo uses; CI (.github/workflows/ci.yml, see the
# go-ci skill) invokes vet / lint / test / test-integration from here so local and
# CI runs stay identical. A new module needs no Makefile changes — `./...` picks it up.

-include .env
export

.PHONY: run build test test-integration test-all lint clean \
       docker-up docker-down migrate-up migrate-down migrate-create \
       migrate-go-up migrate-go-down build-migrate \
       coverage fmt vet swagger

APP_NAME := beer-lms-api
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
	golangci-lint run ./...

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

# --- Swagger ---

swagger:
	swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal

# --- Code quality ---

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

fmt:
	go fmt ./...

vet:
	go vet ./...
