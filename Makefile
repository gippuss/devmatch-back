.PHONY: run test build migrate-up migrate-down docker-up docker-down

run:
	go run ./cmd/app

test:
	go test ./...

build:
	go build ./cmd/app

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down
