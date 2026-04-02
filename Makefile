# コンテナランタイム自動検出 (podman優先)
COMPOSE := $(shell if command -v podman-compose >/dev/null 2>&1; then echo "podman-compose"; elif command -v podman >/dev/null 2>&1; then echo "podman compose"; else echo "docker compose"; fi)
CONTAINER_CMD := $(shell if command -v podman >/dev/null 2>&1; then echo podman; else echo docker; fi)

.PHONY: all
all: format test build lint

.PHONY: generate
generate:
	rm -fr api/ pkg/apis/v1alpha1/api
	go tool ogen apis/v1alpha1/openapi.yaml -clean
	mv api pkg/apis/v1alpha1/

.PHONY: format
format:
	go fmt ./...

# Test commands based on test strategy
.PHONY: test
test:
	go test -v -parallel 4 ./...

# E2E tests (requires running server: make dev-up)
.PHONY: test-e2e
test-e2e:
	go tool runn run e2e/**/*.yaml --verbose
	E2E_BASE_URL=http://localhost:8080 go test -v -count=1 ./e2e/...

# Version detection
VERSION ?= $(shell ./scripts/version.sh)
LDFLAGS := -X github.com/tacokumo/portal-api/pkg/version.Version=$(VERSION)

.PHONY: build
build:
	go build -ldflags "$(LDFLAGS)" -o bin/server ./cmd/server

.PHONY: lint
lint:
	golangci-lint run

# =========================================
# コンテナタスク
# =========================================

.PHONY: dev-up
dev-up: setup-secrets
	$(COMPOSE) up -d

.PHONY: dev-down
dev-down:
	$(COMPOSE) down

.PHONY: dev-logs
dev-logs:
	$(COMPOSE) logs -f

.PHONY: dev-rebuild
dev-rebuild:
	$(COMPOSE) up -d --build

.PHONY: dev-restart
dev-restart:
	$(COMPOSE) restart portal-api

.PHONY: dev-clean
dev-clean:
	$(COMPOSE) down -v --remove-orphans

# =========================================
# 開発環境セットアップタスク
# =========================================

.PHONY: setup-secrets
setup-secrets:
	@echo "Setting up secrets directory..."
	@mkdir -p secrets
	@if [ ! -f secrets/jwt-private-key.pem ]; then \
		echo "Generating JWT key pair..."; \
		./scripts/generate-jwt-keys.sh; \
	else \
		echo "JWT keys already exist"; \
	fi
	@echo "Secrets setup complete"

.PHONY: setup-env
setup-env:
	@if [ ! -f .env ]; then \
		echo "Creating .env file from .env.development.example..."; \
		cp .env.development.example .env; \
		echo "Please edit .env file and set your GitHub OAuth credentials"; \
	else \
		echo ".env file already exists"; \
	fi

.PHONY: setup-dev
setup-dev: setup-env setup-secrets
	@echo "Development environment setup complete!"
	@echo "Next steps:"
	@echo "1. Edit .env file with your GitHub OAuth credentials"
	@echo "2. Run 'make dev-up' to start the development environment"
	@echo "3. Visit http://localhost:8080 to access the API"

# =========================================
# ヘルスチェック・デバッグタスク
# =========================================

.PHONY: health-check
health-check:
	@echo "Checking service health..."
	@curl -f http://localhost:8080/health/liveness || echo "API service is down"
	@$(CONTAINER_CMD) exec portal-valkey valkey-cli ping || echo "Valkey service is down"

.PHONY: debug-logs
debug-logs:
	@echo "=== Portal API Logs ==="
	@$(CONTAINER_CMD) logs portal-api --tail=50
	@echo "=== Valkey Logs ==="
	@$(CONTAINER_CMD) logs portal-valkey --tail=20
