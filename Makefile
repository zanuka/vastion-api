.PHONY: run build test compose-up lint check seed hooks

-include .env
export

run:
	go run ./cmd/api

build:
	go build ./...

test:
	go test ./...

compose-up:
	docker compose up -d

lint:
	golangci-lint run

check: build test lint

seed:
	go run ./cmd/seed

hooks:
	git config core.hooksPath .githooks
