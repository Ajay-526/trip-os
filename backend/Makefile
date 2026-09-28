SHELL := /bin/bash
DATABASE_URL ?= postgres://trip_os:trip_os_dev_password@localhost:5432/trip_os?sslmode=disable

.PHONY: run build test vet migrate-up migrate-down tidy

run:
	go run ./cmd/api

build:
	go build -o bin/tripos-api ./cmd/api

test:
	go test ./... -cover

vet:
	go vet ./...

# Requires golang-migrate (https://github.com/golang-migrate/migrate) CLI
# installed locally: `go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest`
migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down 1

tidy:
	go mod tidy
