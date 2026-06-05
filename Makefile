.PHONY: run dev build tidy db-up db-down migrate-create migrate-up migrate-down migrate-force migrate-version gen-docs fmt test

MIGRATIONS_PATH = ./cmd/migrate/migrations
DB_ADDR ?= postgres://vigil:vigil@localhost:5432/vigil?sslmode=disable

run:
	@go run ./cmd/api

dev:
	@air

build:
	@go build -o bin/api ./cmd/api

tidy:
	@go mod tidy

db-up:
	@docker compose up -d db

db-down:
	@docker compose down

migrate-create:
	@test -n "$(name)" || (echo "usage: make migrate-create name=<snake_case>"; exit 1)
	@migrate create -ext sql -dir $(MIGRATIONS_PATH) -seq $(name)

migrate-up:
	@migrate -path=$(MIGRATIONS_PATH) -database="$(DB_ADDR)" up

migrate-down:
	@migrate -path=$(MIGRATIONS_PATH) -database="$(DB_ADDR)" down 1

migrate-force:
	@test -n "$(version)" || (echo "usage: make migrate-force version=<n>"; exit 1)
	@migrate -path=$(MIGRATIONS_PATH) -database="$(DB_ADDR)" force $(version)

migrate-version:
	@migrate -path=$(MIGRATIONS_PATH) -database="$(DB_ADDR)" version

gen-docs:
	@swag init -g ./api/main.go -d cmd,internal && swag fmt

fmt:
	@go fmt ./... && swag fmt

test:
	@go test -v ./...
