# syntax=docker/dockerfile:1

FROM golang:1.22-alpine AS builder
WORKDIR /src

# Cache module downloads separately from source changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/tripos-api ./cmd/api

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app

COPY --from=builder /out/tripos-api ./tripos-api
COPY migrations ./migrations

# Most PaaS providers (Render, Railway, Fly) inject PORT; internal/config
# falls back to HTTP_PORT=8080 if it's unset.
EXPOSE 8080
ENTRYPOINT ["./tripos-api"]
