.PHONY: proto build build-server build-worker build-webhook build-migrator build-storageui \
	run run-worker run-webhook run-storageui migrate migrate-dry-run \
	test test-integration test-all clean \
	docker-up docker-down docker-clean docker-build docker-logs docker-restart \
	deps cloudbeaver-sync

GO ?= go
PROTOC ?= protoc
CONFIG ?= config.toml
BIN_DIR ?= bin
DOCKER_COMPOSE ?= $(shell if command -v docker-compose >/dev/null 2>&1; then echo docker-compose; else echo "docker compose"; fi)

PROTO_FILES := \
	proto/user.proto \
	proto/tenant.proto \
	proto/release.proto \
	proto/target.proto \
	proto/payload.proto

proto:
	$(foreach file,$(PROTO_FILES),$(PROTOC) --go_out=. --go_opt=module=zxc \
	  --go-grpc_out=. --go-grpc_opt=module=zxc \
	  $(file);)

build: proto build-server build-worker build-webhook build-migrator build-storageui

build-server:
	$(GO) build -o $(BIN_DIR)/server ./cmd/server

build-worker:
	$(GO) build -o $(BIN_DIR)/worker ./cmd/worker

build-webhook:
	$(GO) build -o $(BIN_DIR)/webhook ./cmd/webhook

build-migrator:
	$(GO) build -o $(BIN_DIR)/migrator ./cmd/migrator

build-storageui:
	$(GO) build -o $(BIN_DIR)/storageui ./cmd/storageui

run: build-server
	./$(BIN_DIR)/server -config $(CONFIG)

run-worker: build-worker
	./$(BIN_DIR)/worker -config $(CONFIG)

run-webhook: build-webhook
	./$(BIN_DIR)/webhook -config $(CONFIG)

run-storageui: build-storageui
	./$(BIN_DIR)/storageui -config $(CONFIG)

migrate: build-migrator
	./$(BIN_DIR)/migrator -config $(CONFIG)

migrate-dry-run: build-migrator
	./$(BIN_DIR)/migrator -config $(CONFIG) -dry-run

test:
	$(GO) test -v -short ./internal/... ./cmd/...

test-integration:
	$(GO) test -v -timeout 600s ./test/...

test-all: test test-integration

clean:
	rm -rf $(BIN_DIR)/
	rm -rf api/

docker-up:
	$(DOCKER_COMPOSE) up -d

docker-down:
	$(DOCKER_COMPOSE) down

docker-clean:
	$(DOCKER_COMPOSE) down -v

docker-build:
	$(DOCKER_COMPOSE) build

docker-logs:
	$(DOCKER_COMPOSE) logs -f

docker-restart: docker-down docker-build docker-up

cloudbeaver-sync:
	sh scripts/sync-cloudbeaver-datasources.sh

deps:
	$(GO) mod download
	$(GO) install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	$(GO) install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
