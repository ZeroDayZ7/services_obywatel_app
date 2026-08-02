# --- PROJECT CONFIGURATION ---
DOCKER_COMPOSE_DIR = ./docker
SERVICES_DIR = ./services
PKG_DIR = ./pkg

# Microservices list (folder names inside SERVICES_DIR)
SERVICES = gateway auth-service citizen-docs notification-service audit-service version-service messaging-service

# --- DOCKER SETTINGS ---
TAG = latest
REGISTRY = citizen

.PHONY: help deps-sync deps-upgrade deps-tidy build up down restart update-all updatedeps

help:
	@echo "Platform Management Commands:"
	@echo "  make deps-sync    - Sync workspace and tidy modules"
	@echo "  make deps-upgrade - Upgrade ONLY external libs (no local pkg errors)"
	@echo "  make deps-tidy    - Clean up modules"
	@echo "  make updatedeps   - Upgrade dependencies in all services and run go mod tidy"

# --- SECTION: GO MANAGEMENT ---

updatedeps:
	@echo "🔄 Upgrading dependency versions across all microservices..."
	@for service in $(SERVICES); do \
		if [ -d "$(SERVICES_DIR)/$$service" ]; then \
			echo "--------------------------------------------------"; \
			echo "🚀 Upgrading microservice: $$service"; \
			cd $(SERVICES_DIR)/$$service && \
			go get -u ./... && \
			go mod tidy && \
			cd - > /dev/null; \
		else \
			echo "⚠️ Directory $(SERVICES_DIR)/$$service does not exist, skipping..."; \
		fi \
	done
	@if [ -d "$(PKG_DIR)" ]; then \
		echo "--------------------------------------------------"; \
		echo "🚀 Upgrading shared package: $(PKG_DIR)"; \
		cd $(PKG_DIR) && go get -u ./... && go mod tidy && cd - > /dev/null; \
	fi
	@if [ -f "go.work" ]; then \
		echo "--------------------------------------------------"; \
		echo "🔄 Syncing Go Workspace..."; \
		go work sync; \
	fi
	@echo "✅ All microservices updated successfully!"

deps-sync:
	@echo "🔄 Synchronizing Go workspace..."
	go work sync
	@$(MAKE) deps-tidy

deps-upgrade:
	@echo "⬆️ Upgrading external dependencies (safe mode)..."
	@for service in $(SERVICES); do \
		echo "Upgrading $$service..."; \
		cd $(SERVICES_DIR)/$$service && \
		go mod edit -dropreplace github.com/zerodayz7/platform/pkg && \
		go get -u $$(go list -f '{{if not .Indirect}}{{.Path}}{{end}}' -m all | grep -v "github.com/zerodayz7/platform") || true && \
		go mod tidy && cd - > /dev/null; \
	done
	@echo "Upgrading pkg dependencies..."
	@cd $(PKG_DIR) && go get -u $$(go list -f '{{if not .Indirect}}{{.Path}}{{end}}' -m all | grep -v "github.com/zerodayz7/platform") || true && go mod tidy && cd - > /dev/null
	@go work sync
	@echo "✅ Upgrade finished successfully."

deps-tidy:
	@echo "🧹 Tidying Go modules..."
	@for service in $(SERVICES); do \
		if [ -d "$(SERVICES_DIR)/$$service" ]; then \
			echo "Processing $$service..."; \
			cd $(SERVICES_DIR)/$$service && go mod tidy && cd - > /dev/null; \
		fi \
	done
	@echo "Processing pkg..."
	@cd $(PKG_DIR) && go mod tidy && cd - > /dev/null

# --- SECTION: DOCKER MANAGEMENT ---

build:
	@echo "🚀 Building Docker images..."
	@for service in $(SERVICES); do \
		docker build -t $(REGISTRY)/$$service:$(TAG) $(SERVICES_DIR)/$$service; \
	done

up:
	cd $(DOCKER_COMPOSE_DIR) && docker compose up -d

down:
	cd $(DOCKER_COMPOSE_DIR) && docker compose down

update-all: deps-sync build down up