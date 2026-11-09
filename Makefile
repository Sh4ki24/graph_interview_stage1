APP      := todo-api
BIN      := bin/$(APP)
SEED_BIN := bin/seed
COMPOSE  := docker compose
N        ?= 100          # number of requests for load_test: make load_test N=1000

.DEFAULT_GOAL := help

.PHONY: help build run test test-integration lint-test qa compose-up compose-down seed load_test swag test-qa

## help: list available make targets
help:
	@echo "$(APP) - available commands:"
	@awk '/^## / { \
	  sub(/^## /, ""); \
	  target = substr($$0, 1, index($$0, ":") - 1); \
	  desc = substr($$0, index($$0, ":") + 2); \
	  printf "  \033[36m%-16s\033[0m %s\n", target, desc \
	}' $(MAKEFILE_LIST)

## build: compile the binary into ./bin
build:
	go build -o $(BIN) .
	go build -o $(SEED_BIN) ./cmd/seed

## seed: run the database seeder against the configured database
seed:
	go run ./cmd/seed

## run: run the app locally (starts postgres, regenerates swagger docs, reads config.yaml)
run: swag
	$(COMPOSE) up -d postgres
	@for i in $$(seq 1 30); do \
		$(COMPOSE) exec -T postgres pg_isready -U todo -q && break; sleep 1; done
	go run .

## test: run all unit tests (mocked repository)
test:
	go test ./... -count=1

## test-integration: spin up postgres (localhost:5433), run integration tests against it
test-integration:
	$(COMPOSE) up -d postgres
	@for i in $$(seq 1 30); do \
		$(COMPOSE) exec -T postgres pg_isready -U todo -q && break; sleep 1; done
	TEST_DATABASE_URL=postgres://todo:todo@localhost:5433/todo?sslmode=disable \
		go test -tags integration ./internal/... -count=1 -v
	@$(COMPOSE) rm -sf postgres

## test-qa: run smoke tests against the running API (requires compose-up first)
test-qa:
	go test -tags qa ./qa/... -count=1 -v

## lint-test: lint (golangci-lint, falls back to go vet) then unit tests
lint-test:
	@command -v golangci-lint >/dev/null && golangci-lint run || go vet ./...
	go test ./... -count=1

## qa: lint, unit tests, then integration tests against a real database
qa: lint-test test test-integration

## compose-up: build and run the full stack (app + postgres + prometheus + grafana), regenerates swagger docs first, then runs QA smoke tests
compose-up: swag
	$(COMPOSE) up -d --build
	@echo "Waiting for API to be ready..."
	@for i in $$(seq 1 30); do \
		curl -s -o /dev/null -w '' http://localhost:8080/healthz && break; \
		sleep 1; \
	done
	$(MAKE) test-qa

## compose-down: stop the full stack
compose-down:
	$(COMPOSE) down

## load_test: send N requests (default 100) and capture pprof profiles
load_test:
	./scripts/load_test.sh $(N)

## swag: regenerate swagger docs (needs swag CLI)
swag:
	swag init -g main.go --output docs
