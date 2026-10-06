# --- DB TEST RESET ---
.PHONY:  redis-users rs

redis-users:
	@echo "Lista użytkowników Redis ACL:"
	@docker exec -it cache-redis redis-cli --user redis_root -a "TwojeSuperBazaHaslo123!" ACL USERS

rs: redis-users

