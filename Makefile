# Shortcuts for Mac and Linux. On Windows, run the commands from the README instead.
DATABASE_URL ?= postgres://game:game@localhost:5432/game?sslmode=disable

.PHONY: db run test check

db:
	docker compose up -d --wait

run: db
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/server

test: db
	TEST_DATABASE_URL="$(DATABASE_URL)" go test -race ./...

check:
	test -z "$$(gofmt -l .)"
	go vet ./...
