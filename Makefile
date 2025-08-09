# golang-skeleton - Go microservices monorepo
SHELL := /bin/bash
.DEFAULT_GOAL := help

COMPOSE := docker compose
SERVICE_API_DIR := services/api
SERVICE_WORKER_DIR := services/worker
API_HTTP_ADDR ?= 127.0.0.1:8080
WORKER_HTTP_ADDR ?= 127.0.0.1:8081

-include .env
export

.PHONY: help
help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "}; {printf "\033[36m%-14s\033[0m %s\n", $$1, $$2}'

# --- dependencies -----------------------------------------------------------

.PHONY: tools
tools: ## Install dev tools (Air, golangci-lint)
	go install github.com/air-verse/air@v1.61.1
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0

.PHONY: deps
deps: ## Start only postgres + nats (for local Air development)
	$(COMPOSE) up -d --wait postgres nats

# --- development ------------------------------------------------------------

.PHONY: dev-api
dev-api: deps ## Run api locally with Air hot reload (DB/NATS in Docker)
	SERVICE_NAME=api HTTP_ADDR=$(API_HTTP_ADDR) air -c services/api/.air.toml

.PHONY: dev-worker
dev-worker: deps ## Run worker locally with Air hot reload (NATS in Docker)
	SERVICE_NAME=worker HTTP_ADDR=$(WORKER_HTTP_ADDR) air -c services/worker/.air.toml

.PHONY: ensure-env up up-d
ensure-env:
	@if [ ! -e .env ]; then cp .env.example .env && echo "Created .env from .env.example"; fi

up: ensure-env ## Full dev stack in Docker (Air hot reload inside containers)
	$(COMPOSE) up --build

up-d: ensure-env ## Start the full dev stack in the background
	$(COMPOSE) up --build -d

.PHONY: up-prod
up-prod: ## Prod-like stack from released images, no hot reload
	$(COMPOSE) -f compose.yaml -f compose.prod.yaml up -d --build

.PHONY: down
down: ## Stop the stack (keep volumes)
	$(COMPOSE) down

.PHONY: nuke
nuke: ## Stop the stack and delete data volumes
	$(COMPOSE) down -v

.PHONY: logs
logs: ## Tail logs from the whole stack
	$(COMPOSE) logs -f --tail=100

.PHONY: web-install dev-web web-check
web-install: ## Install React dependencies
	npm ci --prefix services/web

dev-web: ## Run React with Vite HMR (start make dev-api separately)
	npm run dev --prefix services/web

web-check: ## Test and typecheck/build React
	npm test --prefix services/web
	npm run build --prefix services/web

# --- quality ----------------------------------------------------------------

.PHONY: tidy
tidy: ## go mod tidy
	go mod tidy

.PHONY: lint
lint: ## golangci-lint
	golangci-lint run

.PHONY: vet
vet: ## go vet
	go vet ./...

.PHONY: test
test: ## go test (with race detector)
	go test -race -cover ./...

.PHONY: check
check: vet test ## vet + test (CI parity)

# --- demo -------------------------------------------------------------------

.PHONY: demo
demo: ## Send demo requests against running api
	./scripts/demo.sh

# --- kubernetes -------------------------------------------------------------

.PHONY: k8s-diff-dev
k8s-diff-dev: ## Render dev overlay
	kubectl kustomize deploy/k8s/overlays/dev

.PHONY: k8s-apply-dev
k8s-apply-dev: ## Apply dev overlay to current kubectl context
	kubectl apply -k deploy/k8s/overlays/dev

.PHONY: k8s-apply-prod
k8s-apply-prod: ## Apply prod overlay to current kubectl context
	kubectl apply -k deploy/k8s/overlays/prod

# --- docker -----------------------------------------------------------------

.PHONY: docker-build
docker-build: ## Build both service images (release target)
	docker buildx build --target release -t golang-skeleton/api:local -f services/api/Dockerfile .
	docker buildx build --target release -t golang-skeleton/worker:local -f services/worker/Dockerfile .
	docker buildx build --target release -t golang-skeleton/web:local -f services/web/Dockerfile .
