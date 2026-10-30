# Multiplayer Game Backend

Go server for the lobby: players and rooms. Gameplay itself runs through Unity Relay, not this server.

## Run

Needs [Go 1.22+](https://go.dev/dl/) and [Docker Desktop](https://www.docker.com/products/docker-desktop/).

```
docker compose up -d --wait     # start Postgres
go run ./cmd/server             # serve on http://localhost:8080
curl http://localhost:8080/healthz
```

## Docker

The image is what gets deployed. The host sets `PORT` and `DATABASE_URL`.

```
docker build -t game-backend .
docker run -p 8080:8080 -e DATABASE_URL="postgres://..." game-backend
```

## Layout

```
cmd/server/       starts the server
internal/api/     HTTP handlers
internal/store/   SQL queries
migrations/       numbered .sql files, run once each on startup
unity-client/     C# client to copy into Unity
```

## Endpoints

Errors always look like `{"error": "short_code", "message": "..."}`.

| Endpoint | Request | Reply | Errors |
| --- | --- | --- | --- |
| `POST /players` | `{display_name}` | `201 {id, display_name, created_at}` | `400 invalid_display_name` |
| `GET /players/{id}` | | `200 {id, display_name, created_at}` | `400 invalid_player_id`, `404 player_not_found` |
| `POST /rooms` | `{host_player_id, relay_join_code, max_players}` | `201 {id, code, host_player_id, status, relay_join_code, max_players}` | `400 invalid_host_player_id`, `404 player_not_found` |
| `POST /rooms/{code}/join` | `{player_id}` | `200 {code, relay_join_code, host_player_id}` | `400 invalid_player_id`, `404 room_not_found`, `404 player_not_found`, `409 room_closed`, `409 room_full` |
| `POST /rooms/{code}/leave` | `{player_id}` | `204` | `400 invalid_player_id`,  `404 player_not_found`, `404 room_not_found` |
