# Payment Method Health Monitor

## Project Overview
Go + PostgreSQL backend service monitoring payment method health across 6 LATAM countries (MX, BR, CO, AR, CL, PE) with 21 payment methods. Computes metrics, detects zombies/hidden gems/performance alerts, and generates reports.

## Architecture
```
Handler (HTTP/Gin) → Service (business logic) → Repository (SQL/pgx) → PostgreSQL
```
- **No ORM** — raw SQL with pgx parameterized queries
- **No external UUID** — PostgreSQL `gen_random_uuid()`
- Layering: handler → service → repository (never skip layers)

## Key Directories
- `cmd/server/main.go` — Entry point, wires all dependencies
- `internal/handler/` — HTTP handlers (Gin)
- `internal/service/` — Business logic
- `internal/repository/` — SQL queries
- `internal/dto/` — Request/response structs
- `internal/model/` — Domain models
- `internal/database/` — Pool, migrations, seeds
- `internal/templates/` — HTML report template
- `migrations/` — SQL migration files (golang-migrate)
- `seeddata/` — Embedded JSON for country catalog
- `docs/` — Swagger spec

## Development
```bash
cp .env.example .env
make up          # Start Postgres + API (auto-migrate + seed)
make test        # Run all tests with race detector
make test-short  # Skip integration tests
```

## Database
- Port 5434 (mapped from container 5432)
- Credentials in `.env` (default: pmhm/pmhm_secret)
- Migrations run automatically when `AUTO_MIGRATE=true`
- Seed data uses fixed random seed (42) for reproducibility

## Testing
- Integration tests require running Postgres on localhost:5434
- Use `-short` flag to skip integration tests
- Security tests cover SQL injection, malformed JSON, boundary conditions

## API Base URL
`http://localhost:8080`

## Conventions
- All list endpoints use offset pagination (page, page_size params)
- All monetary values in USD (auto-converted via country FX rates)
- Insight detection uses errgroup for concurrency
- Batch transactions are all-or-nothing (single DB transaction)
