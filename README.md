# Multiplayer Game Backend

A Go server for everything outside a match: rooms, run results, the leaderboard.
Live gameplay does not go through this server. Players sync with each other inside Unity.

## Run it

You need [Go 1.22 or newer](https://go.dev/dl/) and [Docker Desktop](https://www.docker.com/products/docker-desktop/).

```
docker compose up -d --wait     # start Postgres
go run ./cmd/server             # start the server on http://localhost:8080
```

Check it is alive:

```
curl http://localhost:8080/healthz
```

## Tests

Tests run against the real database, each in its own temporary schema.

Mac or Linux:

```
make test
```

Windows PowerShell:

```
docker compose up -d --wait
$env:TEST_DATABASE_URL = "postgres://game:game@localhost:5432/game?sslmode=disable"
go test ./...
```

## Where things live

```
cmd/server/main.go      starts the server (finished)
internal/api/           HTTP handlers: read JSON, check it, call the store, write JSON
internal/store/         every SQL query
migrations/             numbered .sql files that build the database
unity-client/           C# files to copy into the Unity project
```

## What to build

| The game needs to | Called by | When |
| --- | --- | --- |
| Open a room and get a short code to share | Host | Before a run |
| Join a room by its code and learn how to reach the host | Other players | Before a run |
| Save a finished run: time survived, score, crew | Host | After a run |
| Read the leaderboard | Everyone | Menu and end screen |
| Read a player's badges | Everyone | Menu |

## Adding an endpoint

1. Decide the request and reply JSON, and write them down in this README.
2. If it needs new tables, add a migration (see `migrations/0000_how_migrations_work.sql`).
3. Add a method on `Store` in `internal/store` with the SQL.
4. Add a handler in `internal/api` and register its route in `api.go`.
5. Add a test in `internal/api/api_test.go`.
6. Add the matching method to `unity-client/BackendClient.cs`.
