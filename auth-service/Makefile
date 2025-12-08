# ===============================
# HTTP-Server Makefile
# ===============================

include .env
include .env.dev

.PHONY: run migrate-up migrate-down migrate-create migrate-goto del-sess build-production

run:
	go build -o $(BIN_DIR)/$(BINARY) $(MAIN_DIR)
	$(BIN_DIR)/$(BINARY)

build-production:
	@echo "Building FreeBSD binary..."
	# uruchomienie skryptu PowerShell w Windows
	powershell -NoProfile -ExecutionPolicy Bypass -File ./scripts/build-freebsd.ps1

migrate-up:
	@echo "Applying all migrations..."
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_PATH)" -verbose up

migrate-down:
	@echo "Rolling back last migration..."
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_PATH)" -verbose down 1

migrate-goto:
	@echo "Podaj numer wersji (np. 1):"; \
	read version; \
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_PATH)" -verbose goto $$version

migrate-create:
	@echo "Podaj nazwę migracji (np. add_column):"; \
	read name; \
	migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $$name

migrate-force:
	@echo "Podaj numer wersji do wymuszenia (np. 0 lub 1):"; \
	read version; \
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_PATH)" force $$version

 
