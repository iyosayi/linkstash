# Linkstash

A small Go + Postgres link shortener service.

It exposes a JSON API to create and fetch short links, plus a redirect endpoint (`GET /{code}`).

## Features

- Create short links (6-letter codes)
- Fetch a link by code
- Redirect by code
- Health check endpoint
- Link “stats” endpoint (schema exists; see note below)

## Requirements

- Go (see `go.mod`)
- Postgres (local or container)
- Optional: `migrate` (golang-migrate) for running SQL migrations

## Configuration

Environment variables:

- `PORT` (default: `8080`)
- `DATABASE_URL` (default: `postgres://postgres:postgres@localhost:5432/linkstash?sslmode=disable`)

## Run locally (with Docker Postgres)

Start Postgres:

```bash
docker compose up -d
```

Apply migrations:

```bash
migrate -path migrations \
  -database "postgres://postgres:postgres@localhost:5432/linkstash?sslmode=disable" \
  up
```

Run the server:

```bash
go run ./cmd/linkstash
```

The server listens on `http://localhost:8080` by default.

## API

### Health check

`GET /healthz`

Response:

```json
{ "status": "ok" }
```

### Create link

`POST /links`

Body:

```json
{ "url": "https://example.com" }
```

Response (201):

```json
{ "code": "abcdef", "url": "https://example.com" }
```

Example:

```bash
curl -sS -X POST "http://localhost:8080/links" \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com"}'
```

### Get link by code

`GET /links/{code}`

Example:

```bash
curl -sS "http://localhost:8080/links/abcdef"
```

### Redirect

`GET /{code}`

Example:

```bash
curl -i "http://localhost:8080/abcdef"
```

### Link stats

`GET /links/{code}/stats`

Response:

```json
{
  "code": "abcdef",
  "url": "https://example.com",
  "click_count": 0,
  "created_at": "2026-05-28T12:34:56Z"
}
```

## Tests

```bash
go test ./...
```

## Database notes

The schema is managed via SQL migrations in `./migrations`.

To open a psql shell against the Docker container:

```bash
docker exec -it linkstash-postgres-1 psql -U postgres -d linkstash
```

