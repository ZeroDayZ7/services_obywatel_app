FROM golang:1.26.2-alpine3.23 AS builder

RUN apk add --no-cache ca-certificates git tzdata

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-w -s" -o /app/auth-service ./cmd/main.go

FROM gcr.io/distroless/static-debian12:latest AS final

LABEL org.opencontainers.image.title="Authentication Service" \
      org.opencontainers.image.description="Professional Auth Service - Distroless Static" \
      org.opencontainers.image.version="1.0.0" \
      org.opencontainers.image.source="https://github.com/ZeroDayZ7/auth-service"

WORKDIR /

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /app/auth-service /auth-service

COPY --from=builder --chown=nonroot:nonroot /src/internal/database/migrations /migrations
COPY --from=builder --chown=nonroot:nonroot /src/logs /logs

USER nonroot:nonroot

EXPOSE 8081

ENTRYPOINT ["/auth-service"]