sqlc generate

go run ./cmd

go get -u ./...
make -C platform updatedeps


docker compose build messaging-service
docker compose build --no-cache messaging-service
docker compose up -d messaging-service

docker compose up --build messaging-service

docker compose up -d --build messaging-service

find . -type f -name "combined.txt" -delete