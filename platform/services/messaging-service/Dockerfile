FROM golang:1.26.2-alpine3.23 AS builder

RUN apk add --no-cache ca-certificates git tzdata

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-w -s" -o /app/citizen-docs ./cmd/main.go

FROM gcr.io/distroless/static-debian12:latest AS final

LABEL org.opencontainers.image.title="Citizen Documents Service" \
      org.opencontainers.image.description="Professional Documents Service - Hardened Static" \
      org.opencontainers.image.version="1.0.0"

WORKDIR /

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /app/citizen-docs /citizen-docs
COPY --from=builder --chown=nonroot:nonroot /src/internal/database/migrations /migrations
COPY --from=builder --chown=nonroot:nonroot /src/logs /logs

USER nonroot:nonroot

EXPOSE 8082

ENTRYPOINT ["/citizen-docs"]