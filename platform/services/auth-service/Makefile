-include .env
-include .env.dev

# Nadpisujemy wartości sztywno pod bezpośrednie połączenie z portem 5450 i kontem root
DB_USER := auth_db_root
DB_PASS := admin
DB_HOST := localhost
DB_PORT := 5450
DB_NAME := $(if $(AUTH_DB_NAME),$(AUTH_DB_NAME),auth_database)
DB_SSLMODE := disable

GOOSE_DRIVER := postgres
GOOSE_DBSTRING := "postgres://$(DB_USER):$(DB_PASS)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)"
MIGRATIONS_DIR := ./migrations

.PHONY: migrate-status migrate-up migrate-down migrate-redo create-migration help

help:
	@echo "Dostępne polecenia migracji Goose dla auth-service:"
	@echo "  make migrate-status  - Sprawdza status wykonanych migracji"
	@echo "  make migrate-up      - Aplikuje wszystkie oczekujące migracje (goose up)"
	@echo "  make migrate-down    - Cofa ostatnią migrację (goose down)"
	@echo "  make migrate-redo    - Cofa i ponownie aplikuje ostatnią migrację"
	@echo "  make create-migration NAME=nazwa - Tworzy nowy plik migracji w ./migrations"

migrate-status:
	@echo "--> Sprawdzanie statusu migracji (target: $(DB_HOST):$(DB_PORT) jako $(DB_USER))..."
	@goose -dir $(MIGRATIONS_DIR) $(GOOSE_DRIVER) $(GOOSE_DBSTRING) status

migrate-up:
	@echo "--> Wykonywanie migracji UP (target: $(DB_HOST):$(DB_PORT) jako $(DB_USER))..."
	@goose -dir $(MIGRATIONS_DIR) $(GOOSE_DRIVER) $(GOOSE_DBSTRING) up

migrate-down:
	@echo "--> Wykonywanie migracji DOWN..."
	@goose -dir $(MIGRATIONS_DIR) $(GOOSE_DRIVER) $(GOOSE_DBSTRING) down

migrate-redo:
	@echo "--> Ponowne wykonywanie ostatniej migracji (redo)..."
	@goose -dir $(MIGRATIONS_DIR) $(GOOSE_DRIVER) $(GOOSE_DBSTRING) redo

create-migration:
	@if [ -z "$(NAME)" ]; then echo "Błąd: Podaj nazwę migracji, np. make create-migration NAME=add_indexes"; exit 1; fi
	@goose -dir $(MIGRATIONS_DIR) create $(NAME) sql