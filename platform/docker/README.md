docker compose up -d
docker compose --profile web up -d
docker compose --profile app up -d
docker compose --profile full up -d

adb -s 5200d78bfa479449 reverse tcp:8080 tcp:8080

docker compose up -d --force-recreate messaging-service 

docker compose --profile "*" logs -f

docker compose --profile "*" logs -f --tail 100
## STRAT ONLY gateway, messaging-service, auth-service, cache-redis
docker compose up -d --force-recreate gateway auth-service messaging-service websocket-service minio

# logs
docker compose logs -f gateway auth-service messaging-service websocket-service minio --tail 100 

docker compose up -d gateway auth-service messaging-service websocket-service minio && docker compose logs -f gateway auth-service messaging-service websocket-service minio --tail 100



docker compose up -d gateway
docker compose up -d messaging-service
docker compose up -d --force-recreate messaging-service



# logs
docker compose logs -f gateway auth-service messaging-service websocket-service minio --tail 100


docker compose logs -f gateway auth-service messaging-service notification-service citizen-docs identity-service officer-bff document-renderer websocket-service --tail 100



# restart -v 
docker compose --profile app down messaging-service db-messaging -v && docker compose --profile app up -d --build db-messaging messaging-service

# ee
docker compose --profile app rm -s -f -v messaging-service db-messaging && docker compose --profile app up -d --build messaging-service db-messaging

docker compose --profile app rm -s -f -v citizen-docs db-citizen-docs && docker compose --profile app up -d --build citizen-docs db-citizen-docs

find . -type f -name "combined.txt" -delete