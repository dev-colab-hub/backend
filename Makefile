.DEFAULT_GOAL := help

# Packages whose tests require Docker (testcontainers).
DOCKER_TEST_PKGS := github.com/sklinkert/go-ddd/internal/infrastructure/db/mongodb \
                    github.com/sklinkert/go-ddd/internal/testhelpers

.PHONY: help build run test test-unit cover lint fmt tidy vendor docker-up docker-down

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

build: ## Build the server binary
	go build -o bin/api ./cmd/api

run: ## Run the server locally
	go run ./cmd/api

test: ## Run all tests with the race detector (needs Docker)
	go test -race ./...

test-unit: ## Run only tests that don't need Docker
	go test -race $(filter-out $(DOCKER_TEST_PKGS),$(shell go list ./...))

cover: ## Run all tests and print total coverage (needs Docker)
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

lint: ## Run golangci-lint
	golangci-lint run ./...

fmt: ## Format code (gofmt + goimports)
	golangci-lint fmt ./...

tidy: ## Tidy and re-vendor modules
	go mod tidy
	go mod vendor

vendor: ## Re-vendor modules
	go mod vendor

docker-up: ## Start MongoDB + app via docker compose
	docker compose up --build

docker-down: ## Stop and remove docker compose resources
	docker compose down -v
