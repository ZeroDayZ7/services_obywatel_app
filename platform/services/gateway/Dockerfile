# Stage 1: Builder
FROM golang:alpine AS builder

RUN apk add --no-cache ca-certificates git tzdata

ENV GOTOOLCHAIN=auto
ENV GOWORK=off

WORKDIR /src

# 1. Kopiujemy pliki workspace oraz go.mod / go.sum dla optymalizacji cache'owania Docker
COPY platform/go.work platform/go.work.sum ./platform/
COPY platform/pkg/go.mod platform/pkg/go.sum ./platform/pkg/
COPY platform/services/gateway/go.mod platform/services/gateway/go.sum ./platform/services/gateway/

# Pobieramy zależności dla usługi
WORKDIR /src/platform/services/gateway
RUN go mod download

# 2. Kopiujemy pełny kod źródłowy obu modułów
WORKDIR /src
COPY platform/pkg ./platform/pkg
COPY platform/services/gateway ./platform/services/gateway

# 3. Kompilujemy aplikację
WORKDIR /src/platform/services/gateway
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-w -s" -o /app/gateway ./cmd/main.go

# Stage 2: Runtime (Distroless)
FROM gcr.io/distroless/static-debian12:latest AS final

LABEL org.opencontainers.image.title="Gateway Service"

WORKDIR /

# Kopiowanie certyfikatów i stref czasowych zbudowanych w builderze
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /app/gateway /gateway

USER nonroot:nonroot
EXPOSE 8085

ENTRYPOINT ["/gateway"]