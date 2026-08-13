.PHONY: run test compose-up lint seed

-include .env
export

run:
	go run ./cmd/api

test:
	go test ./...

compose-up:
	docker compose up -d

lint:
	golangci-lint run

seed:
	go run ./cmd/seed
