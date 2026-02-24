# Payment Method Health Monitor

A Go + PostgreSQL backend service that monitors payment method health across 6 LATAM countries with 21 payment methods. Computes health metrics, detects zombies/hidden gems/performance issues, and surfaces actionable insights.

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────────┐     ┌──────────┐
│  HTTP Client │────▶│   Handler   │────▶│    Service       │────▶│Repository│
│  (curl/UI)   │◀────│ (validation)│◀────│ (business logic) │◀────│  (SQL)   │
└─────────────┘     └─────────────┘     └─────────────────┘     └────┬─────┘
                                                                      │
                                                                ┌─────▼─────┐
                                                                │ PostgreSQL│
                                                                └───────────┘
```

**Layering:**
- **Handlers** (`internal/handler/`): HTTP binding, validation, pagination
- **Services** (`internal/service/`): Business logic, metric computation, insight detection
- **Repositories** (`internal/repository/`): Raw SQL via pgx, domain models
- **DTOs** (`internal/dto/`): Request/response structs
- **Models** (`internal/model/`): Domain structs shared across layers

## Quick Start

```bash
# 1. Clone and setup
git clone https://github.com/anyulbade/payment-method-health-monitor.git
cd payment-method-health-monitor
cp .env.example .env

# 2. Start everything (Postgres + API with auto-migrate + seed data)
make up

# 3. Verify
curl http://localhost:8080/health
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check with DB status |
| POST | `/api/v1/transactions` | Create single transaction (auto USD conversion) |
| POST | `/api/v1/transactions/batch` | Batch insert (max 500, all-or-nothing) |
| GET | `/api/v1/metrics` | Health metrics per payment method/country |
| GET | `/api/v1/insights` | Automated insight detection |
| GET | `/api/v1/trends` | Week/month-over-month trend analysis |
| GET | `/api/v1/roi` | Cost-benefit ROI analysis |
| GET | `/api/v1/market-gaps` | Missing payment method detection |
| GET | `/api/v1/reports/health` | Portfolio health report (JSON or HTML) |
| GET | `/swagger/index.html` | Swagger UI documentation |

## Example Requests

```bash
# Health check
curl http://localhost:8080/health

# Metrics for Brazil (paginated)
curl "http://localhost:8080/api/v1/metrics?country=BR&page=1&page_size=10" | jq .

# All insights
curl "http://localhost:8080/api/v1/insights?page=1&page_size=20" | jq .

# Only zombie insights
curl "http://localhost:8080/api/v1/insights?insight_type=zombie" | jq .

# Month-over-month trends
curl "http://localhost:8080/api/v1/trends?period=MOM&metric=tpv_usd" | jq .

# ROI for Mexico
curl "http://localhost:8080/api/v1/roi?country=MX" | jq .

# Market gaps (essential only)
curl "http://localhost:8080/api/v1/market-gaps?only_essential=true" | jq .

# HTML report
curl "http://localhost:8080/api/v1/reports/health?format=html" -o report.html && open report.html

# Create a transaction
curl -X POST http://localhost:8080/api/v1/transactions \
  -H "Content-Type: application/json" \
  -d '{"payment_method_code":"PIX","country_code":"BR","currency":"BRL","amount":250.50,"status":"APPROVED","transaction_date":"2026-02-01T10:00:00Z"}'

# Batch transactions
curl -X POST http://localhost:8080/api/v1/transactions/batch \
  -H "Content-Type: application/json" \
  -d '{"transactions":[{"payment_method_code":"PIX","country_code":"BR","currency":"BRL","amount":100,"status":"APPROVED","transaction_date":"2026-02-01T10:00:00Z"},{"payment_method_code":"SPEI","country_code":"MX","currency":"MXN","amount":5000,"status":"APPROVED","transaction_date":"2026-02-01T11:00:00Z"}]}'
```

## Insight Detection

### Zombies
Payment methods with active integration costs but very low transaction volume:
- **New methods** (<3 months): flagged if <5 txns in 90 days
- **Established methods**: flagged if dropped below 10% of historical baseline
- Expected: RAPIPAGO (AR), DAVIPLATA (CO), KUESKI (MX), FPAY (CL)

### Hidden Gems
High-performing methods with untapped potential:
- Approval rate >=90%, revenue contribution >=2%, volume share < 75% of revenue share
- Expected: NEQUI (CO), ADDI (CO), YAPE (PE)

### Performance Alerts
Methods underperforming vs. their type-segmented peers:
- Approval rate >10pp below country+type average, with >=20 transactions
- Expected: VISA_CREDIT in MX (60-65% vs ~85% card average)

## Seed Data

~450 transactions across 6 countries (MX, BR, CO, AR, CL, PE) and 21 payment methods over 6 months (Sep 2025 - Feb 2026), with 60% weighted to the last 2 months. Fixed random seed (42) for reproducibility.

**Country Payment Catalog** includes 5 gap methods NOT yet integrated: CODI (MX), BANCOLOMBIA_QR (CO), DEBIN (AR), KHIPU (CL), PLIN (PE).

## Tech Stack

| Library | Purpose |
|---------|---------|
| `gin-gonic/gin` | HTTP router |
| `jackc/pgx/v5` | PostgreSQL driver (pool: MaxConns=25) |
| `golang-migrate/migrate/v4` | Versioned DB migrations |
| `rs/zerolog` | Structured JSON logging |
| `golang.org/x/sync/errgroup` | Concurrent insight detection |
| `stretchr/testify` | Test assertions |
| `html/template` (stdlib) | HTML reports |
| `embed` (stdlib) | Embedded templates + seed data |

No ORM — raw SQL with pgx. No external UUID lib — PostgreSQL `gen_random_uuid()`.

## Design Decisions

1. **Repository/Service/Handler layering** — Enables unit testing services with mock repos, integration testing repos against real DB
2. **golang-migrate** — Versioned, reversible migrations run on startup (dev) or separately (prod)
3. **errgroup for insights** — Zombie, hidden gem, and performance alert detection run concurrently with proper error propagation
4. **All-or-nothing batch** — Batch transaction inserts use a single DB transaction; if any item fails, entire batch rolls back
5. **Offset pagination** — Simple and sufficient for this dataset size; cursor-based would be better at scale
6. **Parameterized queries everywhere** — All user input goes through `$1` placeholders, preventing SQL injection
7. **Embedded templates** — HTML report template compiled into binary via `//go:embed`
8. **Fixed random seed** — Seed data is deterministic and reproducible across runs

## What I'd Improve With More Time

- Cursor-based pagination for better performance at scale
- Redis caching for expensive metric computations
- WebSocket for real-time insight notifications
- Grafana dashboards for trend visualization
- Circuit breaker for external payment provider health checks
- Rate limiting on ingestion endpoints
- OpenTelemetry tracing
- More granular RBAC for API access
- Time-series DB (TimescaleDB) for high-volume transaction data

## Makefile Targets

```bash
make up        # docker-compose up --build -d
make down      # docker-compose down
make test      # go test -race ./... -v
make test-short # go test -short ./... -v (skip integration)
make build     # go build -o bin/server
make metrics   # curl metrics endpoint
make insights  # curl insights endpoint
make trends    # curl trends endpoint
make roi       # curl ROI endpoint
make gaps      # curl market-gaps endpoint
make report    # download and open HTML report
make swagger   # regenerate swagger docs
make clean     # docker-compose down -v + cleanup
```
