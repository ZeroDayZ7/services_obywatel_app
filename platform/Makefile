# --- PROJECT CONFIGURATION ---
DOCKER_COMPOSE_DIR = ./docker
SERVICES_DIR = ./services
PKG_DIR = ./pkg
SERVICES = gateway auth-service citizen-docs notification-service audit-service version-service

# --- DOCKER SETTINGS ---
TAG = latest
REGISTRY = citizen

.PHONY: help deps-sync deps-upgrade deps-tidy build up down restart update-all

help:
	@echo "Platform Management Commands:"
	@echo "  --- GO DEPENDENCIES (Local) ---"
	@echo "  make deps-sync    - Sync workspace and tidy all modules (Docker-independent)"
	@echo "  make deps-upgrade - Upgrade all dependencies to latest minor versions"
	@echo "  make deps-tidy    - Clean up unused modules in services and pkg"
	@echo ""
	@echo "  --- DOCKER INFRASTRUCTURE ---"
	@echo "  make build        - Build Docker images for all services"
	@echo "  make up           - Start infrastructure and containers"
	@echo "  make down         - Stop and remove all containers"
	@echo "  make restart      - Restart the entire stack"
	@echo ""
	@echo "  --- FULL AUTOMATION ---"
	@echo "  make update-all   - Sync, Build, and Redeploy everything"

# --- SECTION: GO MANAGEMENT ---

deps-sync:
	@echo "🔄 Synchronizing Go workspace..."
	go work sync
	@$(MAKE) deps-tidy

deps-upgrade:
	@echo "⬆️ Upgrading all Go dependencies..."
	@for service in $(SERVICES); do \
		echo "Upgrading $$service..."; \
		cd $(SERVICES_DIR)/$$service && go get -u ./... && go mod tidy && cd ../..; \
	done
	@echo "Upgrading shared packages in $(PKG_DIR)..."
	@cd $(PKG_DIR) && go get -u ./... && go mod tidy && cd ..
	@go work sync
	@echo "✅ Upgrade completed."

deps-tidy:
	@echo "🧹 Tidying Go modules..."
	@for service in $(SERVICES); do \
		echo "Processing $$service..."; \
		cd $(SERVICES_DIR)/$$service && go mod tidy && cd ../..; \
	done
	@echo "Processing shared packages..."
	@cd $(PKG_DIR) && go mod tidy && cd ..

# --- SECTION: DOCKER MANAGEMENT ---

build:
	@echo "🚀 Building Docker images..."
	@for service in $(SERVICES); do \
		echo "Building $(REGISTRY)/$$service:$(TAG)..."; \
		docker build -t $(REGISTRY)/$$service:$(TAG) $(SERVICES_DIR)/$$service; \
	done

up:
	@echo "⬆️ Starting infrastructure..."
	cd $(DOCKER_COMPOSE_DIR) && docker compose up -d

down:
	@echo "⬇️ Stopping infrastructure..."
	cd $(DOCKER_COMPOSE_DIR) && docker compose down

restart: down up

# --- SECTION: FULL AUTOMATION ---

update-all: deps-sync build down up
	@echo "🔥 Full platform update successful."