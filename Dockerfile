FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/currency-rate-service ./cmd/server

FROM alpine:3.22

RUN adduser -D -H -u 10001 appuser
WORKDIR /app

COPY --from=builder /out/currency-rate-service ./currency-rate-service
COPY docker/config.yaml ./config.yaml

USER appuser
EXPOSE 8080

ENTRYPOINT ["/app/currency-rate-service"]
