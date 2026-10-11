docker compose up -d
docker compose --profile web up -d
docker compose --profile app up -d
docker compose --profile full up -d

docker compose up -d --force-recreate rabbitmq

docker compose --profile "*" logs -f

docker compose --profile "*" logs -f --tail 100
## STRAT ONLY gateway, messaging-service, auth-service, cache-redis
docker compose up -d gateway auth-service messaging-service db-auth db-messaging cache-redis

# logs
docker compose logs -f gateway auth-service messaging-service --tail 100


docker compose logs -f gateway auth-service messaging-service notification-service citizen-docs identity-service officer-bff document-renderer --tail 100



# restart -v 
docker compose --profile app down messaging-service db-messaging -v && docker compose --profile app up -d --build db-messaging messaging-service

# ee
docker compose --profile app rm -s -f -v messaging-service db-messaging && docker compose --profile app up -d --build messaging-service db-messaging

docker compose --profile app rm -s -f -v citizen-docs db-citizen-docs && docker compose --profile app up -d --build citizen-docs db-citizen-docs