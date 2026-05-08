FROM golang:1.26.2-alpine3.23 AS builder
# Stage 1: Build


RUN apk add --no-cache ca-certificates git tzdata

WORKDIR /src

# 1. Wyłączamy workspace na poziomie zmiennej środowiskowej
ENV GOWORK=off

# 2. Kopiujemy TYLKO wspólny pakiet i moduł gatewaya
# Robimy to przed resztą kodu, żeby go mod download był w osobnym cache'u
COPY platform/pkg/go.mod platform/pkg/go.sum ./platform/pkg/
COPY platform/services/gateway/go.mod platform/services/gateway/go.sum ./platform/services/gateway/

# 3. Pobieramy zależności zewnętrzne (z internetu)
# cd do gateway, bo to on jest naszym "rootem" binarki
RUN cd platform/services/gateway && go mod download

# 4. Kopiujemy faktyczny kod źródłowy tylko tych dwóch folderów
COPY platform/pkg ./platform/pkg
COPY platform/services/gateway ./platform/services/gateway

# 5. Budujemy binarkę
# Ważne: Go sam znajdzie pkg w folderze obok, o ile w go.mod gatewaya 
# masz: replace github.com/ZeroDayZ7/services_obywatel_app/platform/pkg => ../../pkg
RUN cd platform/services/gateway && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-w -s" -o /app/gateway ./cmd/main.go

# Stage 2: Final (Hardened Distroless)
FROM gcr.io/distroless/static-debian12:latest AS final

LABEL org.opencontainers.image.title="Gateway Service" \
      org.opencontainers.image.description="Professional Go API Gateway - Hardened Distroless" \
      org.opencontainers.image.source="https://github.com/ZeroDayZ7/services_obywatel_app"

WORKDIR /

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /app/gateway /gateway

USER nonroot:nonroot
EXPOSE 8081

ENTRYPOINT ["/gateway"]