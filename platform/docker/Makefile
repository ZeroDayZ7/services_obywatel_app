# --- DB TEST RESET ---
.PHONY: reset-db-test rebuild-auth

rdt:
	@echo "Resetowanie bazy i wolumenu db-test..."
	@docker compose stop db-test
	@docker compose rm -f db-test
	@docker volume rm $$(docker compose config --format json | grep -o '"name": "[^"]*' | head -1 | cut -d'"' -f4)_pg_test_data 2>/dev/null || docker volume rm obywatel_app_pg_test_data 2>/dev/null || true
	@docker compose up -d db-test
	@echo "Gotowe! Baza db-test wstała z czystym wolumenem i uruchomiła skrypty init."

rebuild-auth:
	@echo "Zatrzymywanie i przebudowywanie auth-service bez cache..."
	docker compose build --no-cache auth-service
	docker compose up -d --force-recreate auth-service
	@echo "Gotowe! auth-service został przebudowany od zera i uruchomiony."