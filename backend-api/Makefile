.PHONY: test build migrate-up migrate-down

test:
	go test ./...

build:
	go build ./cmd/api

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down
