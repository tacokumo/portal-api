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
# Docker Compose タスク
# =========================================

.PHONY: docker-build
docker-build:
	docker compose build

.PHONY: docker-up
docker-up: setup-secrets
	docker compose up -d

.PHONY: docker-down
docker-down:
	docker compose down

.PHONY: docker-logs
docker-logs:
	docker compose logs -f

.PHONY: docker-clean
docker-clean:
	docker compose down -v --remove-orphans
	docker system prune -f

# 開発環境用タスク
.PHONY: dev-up
dev-up: setup-secrets
	docker compose -f docker-compose.yaml -f docker-compose.dev.yaml up -d

.PHONY: dev-down
dev-down:
	docker compose -f docker-compose.yaml -f docker-compose.dev.yaml down

.PHONY: dev-logs
dev-logs:
	docker compose -f docker-compose.yaml -f docker-compose.dev.yaml logs -f

.PHONY: dev-rebuild
dev-rebuild:
	docker compose -f docker-compose.yaml -f docker-compose.dev.yaml up -d --build

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
	@curl -f http://localhost:8080/healthz || echo "API service is down"
	@docker exec portal-valkey valkey-cli ping || echo "Valkey service is down"

.PHONY: debug-logs
debug-logs:
	@echo "=== Portal API Logs ==="
	@docker logs portal-api --tail=50
	@echo "=== Valkey Logs ==="
	@docker logs portal-valkey --tail=20
