.PHONY: run build test test-integration compose-up lint check seed hooks

-include .env
export

HOOKS_DIR := $(shell git rev-parse --git-path hooks 2>/dev/null)

ifneq ($(HOOKS_DIR),)
$(shell ln -sfn "$(CURDIR)/.githooks/pre-push" "$(HOOKS_DIR)/pre-push")
endif

run:
	go run ./cmd/api

build:
	go build ./...

test:
	go test ./...

test-integration:
	go test -tags=integration ./...

compose-up:
	docker compose up -d

lint:
	golangci-lint run

check: build test lint

seed:
	go run ./cmd/seed

hooks:
	ln -sfn "$(CURDIR)/.githooks/pre-push" "$(HOOKS_DIR)/pre-push"
