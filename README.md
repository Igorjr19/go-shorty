# GO-SHORTY

Simple URL Shortener written in Golang.

This name is no 50 Cent pun. This is Real. Actually it's a pun!

## Quick Start

```bash
# Copy environment variables
cp .env.sample .env

# Start with Docker
make docker-up
make docker-migrate-up
```

Application available at `http://localhost:8080`

## API

### Shorten a URL

`POST /shorten`

```bash
curl -X POST http://localhost:8080/shorten -d '{"url":"https://go.dev"}'
```

`201 Created`

```json
{"code":"aB3xY9","short_url":"http://localhost:8080/aB3xY9"}
```

Errors return a JSON body like `{"error":"URL is required"}`:

| Status | When |
|---|---|
| `400` | Invalid JSON, missing `url`, or `url` is not an absolute `http`/`https` URL |
| `429` | Rate limit exceeded (10 requests/minute per IP) |
| `500` | Unexpected error |

### Resolve a short URL

`GET /{code}` redirects to the original URL with `302 Found` and counts the visit, or returns `404` if the code does not exist. Limited to 1000 requests/minute per IP.

### Stats

`GET /{code}/stats`

```bash
curl http://localhost:8080/aB3xY9/stats
```

`200 OK`

```json
{"code":"aB3xY9","original_url":"https://go.dev","visits":42,"created_at":"2026-09-28T21:00:00Z"}
```

Returns `404` if the code does not exist. Reading stats does not count as a visit.

### Health check

`GET /healthz` returns `{"status":"ok"}` when the database is reachable, or `503` with `{"status":"unavailable"}` otherwise.

## Configuration

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP port |
| `ENVIRONMENT` | `development` | `production` enables JSON logs at info level |
| `BASE_URL` | request host | Base used to build `short_url` |
| `TRUST_PROXY_HEADERS` | `false` | Read client IP from `X-Forwarded-For`/`X-Real-IP`. Only enable behind a trusted reverse proxy |
| `PG*` | | Postgres connection, see `.env.sample` |

## Available Commands

### Docker

```bash
make docker-up           # Start containers
make docker-down         # Stop containers
make docker-logs         # Show logs
make docker-migrate-up   # Apply migrations
make docker-migrate-down # Revert migrations
make docker-clean        # Remove everything
```

### Local

```bash
make run                 # Run application
make migrate-up          # Apply migrations
make migrate-down        # Revert migrations
make test                # Run tests
```

## Tests

```bash
make test
```

Postgres integration tests are skipped unless `TEST_DATABASE_URL` is set:

```bash
make docker-up
TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/shorty?sslmode=disable" make test
```

## Migrations

Format: `NNN_description.{up|down}.sql`

Create new migration:

```bash
# migrations/004_add_expires_at.up.sql
ALTER TABLE links ADD COLUMN expires_at TIMESTAMP;

# migrations/004_add_expires_at.down.sql
ALTER TABLE links DROP COLUMN expires_at;
```

Execute:

```bash
make migrate-up              # All pending
make migrate-up-step STEPS=1 # Only one
```
