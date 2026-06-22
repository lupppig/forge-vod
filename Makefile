BINARY      := forge-vod
WORKER      := worker
BIN_DIR     := bin
COMPOSE     := docker compose

.DEFAULT_GOAL := help

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

.PHONY: services
services: ## Start backing services (postgres, redis, minio) in the background
	$(COMPOSE) up -d postgres redis minio minio-init

.PHONY: services-down
services-down: ## Stop backing services
	$(COMPOSE) down

.PHONY: services-logs
services-logs: ## Tail logs from the backing services
	$(COMPOSE) logs -f

.PHONY: build
build: ## Compile the application binary into ./bin
	go build -o $(BIN_DIR)/$(BINARY) .

.PHONY: run
run: ## Run the application from source
	go run .

.PHONY: worker
worker: ## Run the transcode worker from source
	go run ./cmd/worker

.PHONY: worker-up
worker-up: ## Build and start the worker as a container
	$(COMPOSE) up -d --build worker

.PHONY: worker-logs
worker-logs: ## Tail the worker container logs
	$(COMPOSE) logs -f worker

.PHONY: up
up: services ## Start services then run the application
	go run .

.PHONY: tidy
tidy: ## Sync go.mod/go.sum
	go mod tidy

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: test
test: ## Run tests
	go test ./...

.PHONY: clean
clean: ## Remove build artifacts and stop services with volumes
	rm -rf $(BIN_DIR)
	$(COMPOSE) down -v
