# Ticket Booking Service (Golang)

A production-style ticket booking backend implemented in Go. This repository demonstrates a layered architecture, robust PostgreSQL usage (with transactions and connection pooling), Redis-backed caching with eviction and collection-versioning, and pragmatic concurrency patterns for safe, efficient high-throughput APIs.

---

**Repository**: ticket-booking

**Primary goals**
- Provide a clean, maintainable backend for managing halls, movies, screenings, seats and bookings.
- Showcase backend engineering best-practices: database transactions, migrations, caching strategies, concurrency control, and observability.

---

**Architecture & Project Layout**
- `cmd/api` — application entrypoint and HTTP server wiring (chi router, graceful shutdown).
- `internal/config` — configuration loader (env-driven).
- `internal/logger` — structured logging using `log/slog`.
- `internal/database` — PostgreSQL connection pooling using `pgxpool`.
- `internal/redis` — Redis client bootstrap.
- `internal/auth` — JWT token generation and validation.
- `internal/handler` — HTTP handlers (transport layer).
- `internal/service` — business logic, concurrency & caching orchestration.
- `internal/repository/postgres` — repository layer with SQL queries and transaction helpers.
- `internal/domain` — domain models and repository interfaces.
- `migrations/` — SQL migration files that define the schema.

This separation enforces clear boundaries: handlers orchestrate requests → services implement business rules → repositories perform safe DB operations. Dependency injection is done in `cmd/api/main.go`, wiring repositories, services, cache and auth components.

---

Key Technical Highlights

- Language & ecosystem: Go modules (modern Go), chi router, pgx (PostgreSQL), redis/go-redis, golang-jwt, singleflight from `x/sync`.
- Database pooling & health: `pgxpool` with configurable pool settings and proactive ping checks.
- SQL & Migrations: Declarative SQL migration files present in `migrations/` (up/down), strong use of SQL constraints and error mapping in repositories.
- Transactions & Concurrency Control: Service-level transactions (pgx transactions embedded into `context.Context`) with explicit `FOR UPDATE` locks where needed to avoid race conditions when confirming/cancelling bookings.
- Caching & Cache Invalidation:
  - Redis is used for read-caching of frequently read resources (movies) with a layered strategy: resource keys (e.g. `movies:<id>`) and collection-versioned keys (e.g. `movies:v<version>:all`).
  - Collection-versioning allows cheap, atomic invalidation of entire collections by incrementing a version pointer (master key), avoiding expensive multi-key deletes.
  - Explicit eviction for single-resource changes (synchronous `DEL`) combined with version increments for collection updates.
  - Dynamic TTL and short-caching of empty results to protect against cache penetration.
  - Background (fire-and-forget) cache writes to avoid blocking request paths.
- Thundering Herd protection: `singleflight.Group` is used to ensure one upstream database query when a cache miss happens concurrently across requests.
- Defensive context usage: database operations use bounded timeouts detached from the caller when executing leader-only operations inside singleflight groups.

---

Database & Transactions (detailed)

- Repositories accept a context that may contain a `pgx.Tx` transaction (helper in `internal/repository/postgres/dbtx.go`).
- Services open explicit transactions via `pgxpool.Pool.Begin`, embed the transaction in context, and defer `Rollback` with a final `Commit` on success — this ensures atomic multi-step operations such as placing a hold and confirming a booking.
- Pessimistic locking (`SELECT ... FOR UPDATE`) is used before updates to ticket rows to prevent concurrent status transitions (e.g., double-booking or cancelling/confirming race conditions).

---

Caching strategy (detailed)

- Per-resource cache: `movies:<id>` stores serialized JSON of a resource for fast reads and avoids repeated DB hits.
- Collection versioning: `movies:version` is incremented whenever collection-affecting changes occur (create, update, delete). List endpoints use `movies:v<version>:all` keys that implicitly change when the version increments.
- Eviction: Specific keys are `DEL`-ed synchronously for immediate consistency of single-resource endpoints; collection changes increment the version pointer to invalidate all list caches cheaply.
- Cache write strategy: On cache miss, the service uses `singleflight` to run exactly one DB fetch, then writes to Redis asynchronously in a background goroutine with a bounded context.
- TTL and empty-set protection: Empty results are cached with a shorter TTL to avoid continuous DB pressure from missing data.

---

Concurrency & Performance Patterns

- Use of `golang.org/x/sync/singleflight` to suppress duplicate concurrent database calls for the same cache key.
- Non-blocking cache updates (goroutines) to keep p99 latency low.
- Connection pooling configuration to control max/min connections and connection lifecycles for predictable DB resource usage.

---

Security & Auth

- JWT-based authentication with HMAC signing, token generation and validation helpers in `internal/auth`.
- Protected routes are mounted using auth middleware in `cmd/api/main.go`.

---

Developer Notes — How to run (local)

1. Set environment variables (see `internal/config` for names expected): DB credentials, Redis host/port, JWT secret, app port, etc.
2. Apply migrations in `migrations/` to create schema (uses plain SQL files in the repo).
3. Start the API (example):

```bash
go run ./cmd/api
```

The server uses a graceful shutdown sequence and exposes a `/health` endpoint for readiness checks.

---

What recruiters will see — skills demonstrated in this project

- Advanced Golang: idiomatic packages, dependency injection via constructors, context propagation, and use of `singleflight` and goroutines for concurrency control.
- Distributed caching: Redis setup, cache key design, TTL strategies, background refresh, explicit eviction, and collection-version invalidation patterns.
- Database engineering: PostgreSQL schema migrations, `pgxpool` connection pooling, safe use of transactions, `SELECT ... FOR UPDATE` locking, robust error mapping for constraint violations.
- System design: layered architecture (handlers/services/repositories), graceful shutdown, health checks, and configuration by environment.
- Security: JWT-based auth and middleware integration.
- Observability & Reliability: structured logging with `log/slog`, bounded contexts/timeouts for DB and cache operations, and defensive coding to avoid thundering-herd and cache-stampede.

---

Notable files & where to look
- Application bootstrap: [cmd/api/main.go](cmd/api/main.go)
- Database pool: [internal/database/postgres.go](internal/database/postgres.go)
- Redis client: [internal/redis/redis.go](internal/redis/redis.go)
- Movie caching + concurrency: [internal/service/movie.go](internal/service/movie.go)
- Booking transactions: [internal/service/booking.go](internal/service/booking.go)
- Repository transaction helper: [internal/repository/postgres/dbtx.go](internal/repository/postgres/dbtx.go)
- SQL migrations: [migrations/](migrations)

---

If you'd like, I can also:
- add a concise CONTRIBUTING.md or CODE_OF_CONDUCT.md
- generate a minimal OpenAPI spec for the public endpoints
- add GitHub Actions workflow for building and linting

---

Author

This repository showcases backend engineering skills and pragmatic production patterns in Go.
