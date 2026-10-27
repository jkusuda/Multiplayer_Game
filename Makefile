# Shortcuts for Mac and Linux. On Windows, run the commands from the README.
DATABASE_URL ?= postgres://game:game@localhost:5432/game?sslmode=disable

.PHONY: db run check

db:
	docker compose up -d --wait

run: db
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/server

check:
	test -z "$$(gofmt -l .)"
	go vet ./...
