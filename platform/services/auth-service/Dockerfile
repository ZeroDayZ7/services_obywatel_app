# Stage 1: Build
FROM golang:1.26.2-alpine3.23 AS builder

RUN apk add --no-cache ca-certificates git tzdata

WORKDIR /src

# Wyłączamy workspace, aby Go korzystał z mechanizmu 'replace' zdefiniowanego w go.mod
ENV GOWORK=off

# 1. Kopiujemy pliki go.mod i go.sum (optymalizacja cache warstw)
# Musimy mieć oba moduły, bo auth-service zależy od pkg
COPY platform/pkg/go.mod platform/pkg/go.sum ./platform/pkg/
COPY platform/services/auth-service/go.mod platform/services/auth-service/go.sum ./platform/services/auth-service/

# 2. Pobieramy zależności
# Robimy to w katalogu serwisu, Go sam rozwiąże lokalne zależności przez 'replace'
RUN cd platform/services/auth-service && go mod download

# 3. Kopiujemy faktyczny kod źródłowy
COPY platform/pkg ./platform/pkg
COPY platform/services/auth-service ./platform/services/auth-service

# 4. Budujemy binarkę
# -trimpath usuwa ścieżki z systemu budującego (bezpieczeństwo)
# -ldflags="-w -s" odchudza binarkę (usuwa debug info)
RUN cd platform/services/auth-service && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-w -s" -o /app/auth-service ./cmd/main.go

# Stage 2: Final (Hardened Distroless)
# Używamy static-debian12, bo nie potrzebujemy żadnych bibliotek dynamicznych (CGO_ENABLED=0)
FROM gcr.io/distroless/static-debian12:latest AS final

LABEL org.opencontainers.image.title="Auth Service" \
      org.opencontainers.image.description="Professional Auth Service - Hardened Distroless" \
      org.opencontainers.image.source="https://github.com/ZeroDayZ7/services_obywatel_app"

WORKDIR /

# Kopiujemy certyfikaty i strefy czasowe z buildera
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Kopiujemy binarkę
COPY --from=builder /app/auth-service /auth-service

# Migracje bazy danych (ważne dla Autha!)
# Distroless nie ma shella, więc binarka musi sama wiedzieć jak je odczytać z tej ścieżki
COPY --from=builder --chown=nonroot:nonroot /src/platform/services/auth-service/internal/database/migrations ./migrations

USER nonroot:nonroot

# Port dla auth-service to 8082
EXPOSE 8082

ENTRYPOINT ["/auth-service"]