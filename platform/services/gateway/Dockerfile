FROM golang:1.26.2-alpine3.23 AS builder

RUN apk add --no-cache ca-certificates git tzdata
WORKDIR /src
ENV GOWORK=off

COPY platform/pkg/go.mod platform/pkg/go.sum ./platform/pkg/
COPY platform/services/gateway/go.mod platform/services/gateway/go.sum ./platform/services/gateway/

RUN cd platform/services/gateway && go mod download

COPY platform/pkg ./platform/pkg
COPY platform/services/gateway ./platform/services/gateway

RUN cd platform/services/gateway && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-w -s" -o /app/gateway ./cmd/main.go

FROM gcr.io/distroless/static-debian12:latest AS final

LABEL org.opencontainers.image.title="Gateway Service"

WORKDIR /
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /app/gateway /gateway

USER nonroot:nonroot
EXPOSE 8081

ENTRYPOINT ["/gateway"]