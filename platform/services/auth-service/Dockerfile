FROM golang:1.26.2-alpine3.23 AS builder

RUN apk add --no-cache ca-certificates git tzdata
WORKDIR /src
ENV GOWORK=off

COPY platform/pkg/go.mod platform/pkg/go.sum ./platform/pkg/
COPY platform/services/auth-service/go.mod platform/services/auth-service/go.sum ./platform/services/auth-service/

RUN cd platform/services/auth-service && go mod download

COPY platform/pkg ./platform/pkg
COPY platform/services/auth-service ./platform/services/auth-service

RUN cd platform/services/auth-service && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-w -s" -o /app/auth-service ./cmd/main.go

FROM gcr.io/distroless/static-debian12:latest AS final

LABEL org.opencontainers.image.title="Auth Service"

WORKDIR /
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /app/auth-service /auth-service

USER nonroot:nonroot
EXPOSE 8082

ENTRYPOINT ["/auth-service"]