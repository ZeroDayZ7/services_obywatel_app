docker compose up -d
docker compose --profile web up -d
docker compose --profile app up -d
docker compose --profile full up -d

docker compose up -d --force-recreate rabbitmq

docker compose --profile "*" logs -f

docker compose --profile "*" logs -f --tail 100

docker compose logs -f gateway auth-service messaging-service notification-service citizen-docs identity-service officer-bff document-renderer



docker compose --profile app rm -s -f -v messaging-service db-messaging && docker compose --profile app up -d --build messaging-service db-messaging

docker compose --profile app rm -s -f -v citizen-docs db-citizen-docs && docker compose --profile app up -d --build citizen-docs db-citizen-docs