-include .env

.PHONY: up down logs build test tidy migrate-up migrate-down

up:
	docker compose up -d --build

down:
	docker compose down

logs:
	docker compose logs -f api

build:
	docker build -t web3-backend-api .

test:
	docker run --rm -v "$(PWD)":/app -w /app golang:1.22-alpine go test ./...

tidy:
	docker run --rm -v "$(PWD)":/app -w /app golang:1.22-alpine go mod tidy

migrate-up:
	docker run --rm --network web3-backend_default -v "$(PWD)/migrations":/migrations migrate/migrate -path=/migrations -database "$(DATABASE_URL)" up

migrate-down:
	docker run --rm --network web3-backend_default -v "$(PWD)/migrations":/migrations migrate/migrate -path=/migrations -database "$(DATABASE_URL)" down 1
