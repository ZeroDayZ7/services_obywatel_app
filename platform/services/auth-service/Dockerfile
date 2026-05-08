# Stage 1: Build
FROM golang:1.26.2-alpine3.23 AS builder

RUN apk add --no-cache ca-certificates git tzdata

WORKDIR /src

# Wyłączamy workspace, aby wymusić mechanizm 'replace' w go.mod
ENV GOWORK=off

# 1. Kopiujemy go.mod/sum dla pkg oraz auth-service (optymalizacja cache'u)
COPY platform/pkg/go.mod platform/pkg/go.sum ./platform/pkg/
COPY platform/services/auth-service/go.mod platform/services/auth-service/go.sum ./platform/services/auth-service/

# 2. Pobieramy zależności
RUN cd platform/services/auth-service && go mod download

# 3. Kopiujemy źródła
COPY platform/pkg ./platform/pkg
COPY platform/services/auth-service ./platform/services/auth-service

# 4. Budujemy binarkę
RUN cd platform/services/auth-service && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-w -s" -o /app/auth-service ./cmd/main.go

# Stage 2: Final (Hardened Distroless)
FROM gcr.io/distroless/static-debian12:latest AS final

LABEL org.opencontainers.image.title="Authentication Service" \
      org.opencontainers.image.description="Professional Auth Service - Hardened Distroless" \
      org.opencontainers.image.source="https://github.com/ZeroDayZ7/services_obywatel_app"

WORKDIR /

# Zasoby systemowe
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Binarka aplikacji
COPY --from=builder /app/auth-service /auth-service

# Migracje bazy danych - kopiujemy tylko jeśli faktycznie istnieją w repo
# Distroless nie ma shella, więc pliki muszą być na miejscu
COPY --from=builder --chown=nonroot:nonroot /src/platform/services/auth-service/internal/database/migrations ./migrations

# Folder /logs USUNIĘTY - logger pisze na stdout (Wazuh/K3s)

USER nonroot:nonroot
EXPOSE 8081

ENTRYPOINT ["/auth-service"]