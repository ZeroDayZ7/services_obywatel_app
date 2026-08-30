# Stage 1: Builder
FROM golang:alpine AS builder

RUN apk add --no-cache ca-certificates git tzdata

ENV GOTOOLCHAIN=auto

ENV GOWORK=off

WORKDIR /src

# 1. Kopiujemy pliki workspace oraz go.mod / go.sum dla optymalizacji cache'owania Docker
COPY platform/go.work platform/go.work.sum ./platform/
COPY platform/pkg/go.mod platform/pkg/go.sum ./platform/pkg/
COPY platform/services/auth-service/go.mod platform/services/auth-service/go.sum ./platform/services/auth-service/

# Pobieramy zależności dla usługi
WORKDIR /src/platform/services/auth-service
RUN go mod download

# 2. Kopiujemy pełny kod źródłowy obu modułów
WORKDIR /src
COPY platform/pkg ./platform/pkg
COPY platform/services/auth-service ./platform/services/auth-service

# 3. Kompilujemy aplikację z wykorzystaniem go.work z katalogu /src/platform
WORKDIR /src/platform/services/auth-service
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-w -s" -o /app/auth-service ./cmd/main.go

# Stage 2: Runtime (Distroless)
FROM gcr.io/distroless/static-debian12:latest AS final

LABEL org.opencontainers.image.title="Auth Service"

WORKDIR /

# Kopiowanie certyfikatów i stref czasowych zbudowanych w builderze
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /app/auth-service /auth-service

USER nonroot:nonroot
EXPOSE 8082

ENTRYPOINT ["/auth-service"]