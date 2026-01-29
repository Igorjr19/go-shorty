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

## Migrations

Formato: `NNN_description.{up|down}.sql`

Create new migration:

```bash
# migrations/002_add_visits.up.sql
ALTER TABLE links ADD COLUMN visits INT DEFAULT 0;

# migrations/002_add_visits.down.sql
ALTER TABLE links DROP COLUMN visits;
```

Execute:

```bash
make migrate-up              # All pending
make migrate-up-step STEPS=1 # Only one
```
