FROM golang:1.27-alpine AS builder

WORKDIR /app
COPY go.work go.work.sum ./
COPY pkg ./pkg
COPY services/websocket-service ./services/websocket-service

WORKDIR /app/services/websocket-service
RUN go build -o /bin/websocket-service ./cmd/main.go

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /bin/websocket-service /bin/websocket-service
CMD ["/bin/websocket-service"]
