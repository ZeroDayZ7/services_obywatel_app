# --- DB TEST RESET ---
.PHONY: reset-db-test

rdt:
	@echo "Resetowanie bazy i wolumenu db-test..."
	@docker compose stop db-test
	@docker compose rm -f db-test
	@docker volume rm $$(docker compose config --format json | grep -o '"name": "[^"]*' | head -1 | cut -d'"' -f4)_pg_test_data 2>/dev/null || docker volume rm obywatel_app_pg_test_data 2>/dev/null || true
	@docker compose up -d db-test
	@echo "Gotowe! Baza db-test wstała z czystym wolumenem i uruchomiła skrypty init."