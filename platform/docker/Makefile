# --- DB TEST RESET ---
.PHONY:  redis-users rs start-services st logs res

redis-users:
	@echo "Lista użytkowników Redis ACL:"
	@docker exec -it cache-redis redis-cli --user redis_root -a "TwojeSuperBazaHaslo123!" ACL USERS

rs: redis-users

start-services:
	docker compose up -d gateway auth-service messaging-service websocket-service minio
	docker compose logs -f gateway auth-service messaging-service websocket-service --tail 100

st: start-services

logs: 
	docker compose logs -f gateway auth-service messaging-service websocket-service --tail 100

res:
	docker compose --profile app down messaging-service db-messaging -v && docker compose --profile app up -d --build db-messaging messaging-service

