docker compose up -d
docker compose --profile web up -d
docker compose --profile app up -d
docker compose --profile full up -d

docker compose up -d --force-recreate rabbitmq

docker compose --profile "*" logs -f

docker compose --profile "*" logs -f --tail 100