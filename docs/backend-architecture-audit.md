# Backend Architecture Audit — Somotracker

**Date:** 2025-08-25  
**Auditor:** AI Assistant  
**Scope:** Complete backend (`/backend` directory)

---

## Executive Summary

The Somotracker backend is a well-structured, production-ready Go application built on Fiber (HTTP), pgx (PostgreSQL), Redis, Asynq (background jobs), and Uber fx (DI). It follows **Functional Domain Package Layering** with strict isolation, canonical error handling, Row-Level Security (RLS) for multi-tenancy, and a per-request transaction boundary that scopes `app.current_tenant_id` via a GUC.

**Overall assessment:** **Scalable for moderate load (10–50 req/s per instance)** but has several architectural bottlenecks that will limit horizontal scaling and increase operational risk under higher concurrency.

---

## 1. Architecture Overview

| Layer | Technology | Notes |
|-------|------------|-------|
| HTTP | Fiber v2 | Custom error handler, body limit 4 MB, timeouts set |
| DI | Uber fx | Module-per-domain, `fx.Annotate` with multiple `fx.As` |
| DB | pgx/v5 + pgxpool | `MaxConns=25`, `MinConns=5`, RLS via `SET LOCAL app.current_tenant_id` |
| Cache/Queue | Redis (go-redis v9) + Asynq | Session cache, rate limiting, background jobs |
| Auth | Stytch (magic links) + cookie sessions | Device fingerprinting (C5), school-scoped cookies (C2) |
| Logging | Zap (sugared) | Structured, request-ID correlation |
| Telemetry | Pluggable sinks (ZapSink default) | Error policies per status class |
| Migrations | golang-migrate | Runs on startup via fx `OnStart` |

### Request Flow (Authenticated)

```
Fiber → WithLogger → PanicRecover → RequestID → CORS → SecurityHeaders
  → CSRF → IP RateLimiter → DeviceFingerprinter → SessionResolver
  → WithTenantContext (BEGIN tx, SET LOCAL app.current_tenant_id)
  → AccessLog → User RateLimiter → Handler → Commit/Rollback
```

---

## 2. Scalability Analysis

### 2.1 Critical Bottlenecks

| # | Component | Issue | Impact | Severity |
|---|-----------|-------|--------|----------|
| **S-1** | **Per-request transaction** (`WithTenantContext`) | Every authenticated request opens a `pgx.Tx` for the **entire request lifecycle** (15–30 s timeouts). Holds a pool connection, blocks other requests. | **High** — Pool exhaustion at ~25 concurrent authenticated requests; latency spikes under load. |
| **S-2** | **Fixed pool size (25)** | `MaxConns=25` hardcoded in `database.Connect`. No config, no auto-tuning. | **High** — Cannot scale vertically; 25 connections = hard ceiling per instance. |
| **S-3** | **Single Asynq worker (concurrency=1)** | `attendance.Worker` runs with `Concurrency: 1`, `Queues: {"summaries": 10}`. All summary refreshes serialize. | **Medium** — Summary recomputation backlog grows linearly with schools/terms. |
| **S-4** | **Imports worker (concurrency=3)** | `importsWorker.Concurrency = 3`. Max 3 chunks in parallel; `ChunkSize=100`, `MaxImportRows=5000` → 50 chunks, ~17 sequential batches. | **Medium** — Large imports take minutes; blocks queue for other tenants. |
| **S-5** | **No read replicas / read-write split** | All queries (including heavy analytics) hit primary. | **Medium** — Write contention under reporting load. |
| **S-6** | **No pgBouncer / connection pooling proxy** | Direct pgxpool per instance; each instance holds 25 connections. | **Medium** — Connection churn on deploy/scale; no transaction pooling. |
| **S-7** | **Migrations run on every instance startup** | `RunMigrations` in `fx.OnStart`. Multiple replicas → concurrent `migrate.Up()` races. | **Low/Medium** — Lock contention, potential deadlocks on deploy. |
| **S-8** | **Redis client unconfigured** | Default pool (10 connections), no timeouts, no retry policy. | **Low** — Redis saturation under session/rate-limit load. |
| **S-9** | **No circuit breaker for Stytch** | Auth calls to Stytch have no timeout/breaker; downstream latency propagates. | **Low** — Cascading failures during Stytch incidents. |
| **S-10** | **No observability (metrics/tracing)** | No Prometheus, OpenTelemetry, or health-check dependencies. | **Low** — Blind spots for capacity planning. |

### 2.2 Connection Pool Math

| Parameter | Value | Notes |
|-----------|-------|-------|
| `MaxConns` | 25 | Hardcoded |
| `MinConns` | 5 | |
| Request hold time | ~50–200 ms (typical) | But up to 15 s (ReadTimeout) |
| Max theoretical throughput | ~125 req/s (at 200 ms) | **Only if no long requests** |
| Realistic safe concurrency | **15–20** | Headroom for slow queries, GC, network |

> **Rule of thumb:** With per-request transactions, treat `MaxConns` as **max concurrent authenticated requests**. At 25, a single slow query (e.g., heavy report) blocks the pool.

### 2.3 Background Job Throughput

| Worker | Concurrency | Queue | Estimated throughput |
|--------|-------------|-------|---------------------|
| Attendance summaries | 1 | summaries | ~1 job/s (heavy SQL) |
| Imports chunk processing | 3 | imports | ~300 rows/s (100 rows/chunk × 3) |
| Cleanup scheduler | N/A | (scheduled) | 1/day |

**Implication:** A school with 500 students × 10 learning areas × 3 terms = 15,000 summary rows per refresh. Single-threaded worker = minutes per school.

---

## 3. Architectural Strengths

| Area | Observation |
|------|-------------|
| **Domain isolation** | Strict package boundaries; no circular imports; cross-domain via orchestrator services or DB views. |
| **RLS + GUC** | `app.current_tenant_id` set per transaction; DB enforces tenant isolation even if app bugs. |
| **Error handling** | Canonical `{code, message, errors}` JSON; single `HTTPError` mapping; structured `xerrors.DomainError`. |
| **Session security** | Redis cache + singleflight + negative cache; device fingerprint (v2, no IP); cookie-scoped school (C2). |
| **Rate limiting** | Two-tier (IP coarse + user fine); Stytch callbacks exempted (C3). |
| **DI discipline** | One `fx.Annotate` per constructor; multiple `fx.As` for interfaces; no globals. |
| **Testing** | Unit (`*_service_test.go` with mocks) + Integration (`*_repository_test.go` with live PG). |
| **Migrations** | Versioned, reversible, RLS policies included. |
| **Telemetry** | Pluggable sinks, error policies per status class, enrichment helpers. |

---

## 4. Detailed Findings & Recommendations

### 4.1 Database & Connection Management

#### Finding S-1: Per-request transaction holds connection for full request

**File:** `internal/middleware/tenantcontext.go:44-84`  
**Code pattern:**
```go
tx, err := pools.PG.Begin(ctx)
// ... set GUC ...
c.SetUserContext(ctx) // ctx carries tx
// handler runs, may call external services
if c.Response().StatusCode() >= 400 {
    tx.Rollback(...)
} else {
    tx.Commit(...)
}
```

**Problem:** A `GET /api/v1/students/list` that takes 200 ms holds a connection for 200 ms. A slow report (5 s) holds it for 5 s. 25 such requests = pool exhausted.

**Recommendations:**
1. **Short-term:** Add `pgx.TxOptions{AccessMode: pgx.ReadOnly}` for read-only routes (detect via method/route). Still needs transaction for `SET LOCAL`, but read-only tx avoids write locks.
2. **Medium-term:** Move tenant scoping to **connection checkout** using pgxpool `AfterConnect` hook:
   ```go
   pgCfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
       // tenant_id not known here — need per-request
   }
   ```
   Instead, use a **connection pool wrapper** that sets `app.current_tenant_id` on `Acquire` and resets on `Release`. Requires pgxpool v5 `Acquire` hook (not yet available) or a custom pool.
3. **Long-term:** Adopt **pgBouncer in transaction pooling mode**. Application uses `SET LOCAL app.current_tenant_id` inside a short explicit transaction only for writes; reads use a separate read-only pool without transaction.

#### Finding S-2: Hardcoded pool size

**File:** `internal/database/database.go:18-20`
```go
pgCfg.MaxConns = 25
pgCfg.MinConns = 5
```

**Recommendation:** Make configurable via `Config` (e.g., `DB_MAX_CONNS`, `DB_MIN_CONNS`). Default to `runtime.NumCPU() * 4` (capped at 100). Document pgBouncer sizing.

#### Finding S-8: Redis client unconfigured

**File:** `internal/database/database.go:45-52`
```go
opt, _ := redis.ParseURL(cfg.RedisURL)
rdb := redis.NewClient(opt) // uses defaults: PoolSize=10, no timeouts
```

**Recommendation:** Configure pool size, timeouts, retry policy:
```go
rdb := redis.NewClient(&redis.Options{
    Addr:         opt.Addr,
    Password:     opt.Password,
    DB:           opt.DB,
    TLSConfig:    opt.TLSConfig,
    PoolSize:     cfg.RedisPoolSize,       // e.g., 50
    MinIdleConns: cfg.RedisMinIdleConns,   // e.g., 10
    DialTimeout:  5 * time.Second,
    ReadTimeout:  3 * time.Second,
    WriteTimeout: 3 * time.Second,
    MaxRetries:   3,
})
```

---

### 4.2 Background Job Processing

#### Finding S-3: Attendance worker concurrency = 1

**File:** `internal/attendance/worker.go:218-224`
```go
w.server = database.NewAsynqServer(w.pools, w.logger, asynq.Config{
    Concurrency: 1,
    Queues:      map[string]int{"summaries": 10},
})
```

**Problem:** All summary refresh tasks (teacher delivery, workload, attendance term, class daily, class learning area, class term) serialize. A single large school can block all others.

**Recommendations:**
1. Increase `Concurrency` to `runtime.NumCPU()` (e.g., 4–8).
2. Split queues by priority: `summaries_high` (teacher delivery), `summaries_low` (class rollups).
3. Add **per-tenant rate limiting** inside workers to prevent one tenant monopolizing workers.
4. Consider **partitioning by tenant** (separate queues) if multi-tenancy grows.

#### Finding S-4: Imports worker concurrency = 3

**File:** `internal/imports/module.go:38-46`
```go
var importsServerConfig = asynq.Config{
    Concurrency: 3,
    Queues: map[string]int{"imports": 10},
}
```

**Problem:** `ChunkSize=100`, `MaxImportRows=5000` → 50 chunks. At 3 workers, ~17 sequential batches. A 5000-row import takes minutes; other tenants' imports wait.

**Recommendations:**
1. Increase concurrency (e.g., 8–16) and use `asynq.Queue("imports")` weight to prioritize.
2. Add **priority queue** for small imports (< 100 rows) → process immediately.
3. Implement **tenant fair-queuing**: track in-flight chunks per tenant, deprioritize if > N.

---

### 4.3 Multi-Tenancy & RLS

#### Finding M-1: RLS policies may lack optimal indexes

**Evidence:** Migration `000062_row_level_security.up.sql` enables RLS on many tables but indexes on `tenant_id` are not explicitly verified.

**Risk:** Sequential scans on large tenant tables → high CPU, slow queries.

**Action:** Audit every RLS-protected table for composite indexes matching policy predicates (e.g., `(tenant_id, school_id, academic_term_id)`). Add `EXPLAIN ANALYZE` checks in CI.

#### Finding M-2: `fn_resolve_session` is `SECURITY DEFINER` (bypasses RLS)

**File:** `internal/middleware/sessionresolver.go:317-352` calls `fn_resolve_session($1)` which is `SECURITY DEFINER`.

**Risk:** If function has SQL injection or logic bug, it can read cross-tenant sessions.

**Mitigation:** Function is simple (lookup by token_hash). Ensure it's audited and immutable. Consider moving session resolution to application layer (already cached in Redis) and remove `SECURITY DEFINER` if possible.

---

### 4.4 Security & Auth

| Finding | File | Issue | Fix |
|---------|------|-------|-----|
| **SEC-1** | `config.go:58-60` | Default `DATABASE_URL` contains password `somo_secure_password` in source. | Remove default; require env var in all envs. |
| **SEC-2** | `config.go:62-64` | Default `COOKIE_SECRET = "dev-insecure-change-in-production"`. Panics in non-dev if unchanged. | OK (fails fast), but remove default entirely. |
| **SEC-3** | `sessionresolver.go:185-210` | Device fingerprint v2 uses `User-Agent + Accept-Language` only (no IP). Legitimate but weak. | Acceptable per C5 design; document trade-off. |
| **SEC-4** | `tenantcontext.go:66-72` | On error response (status >= 400), rolls back tx **but does not reset GUC**. Next request on same connection (if pool reuses) may inherit stale `app.current_tenant_id`? | `SET LOCAL` is transaction-scoped; rollback resets it. Safe. |
| **SEC-5** | `register.go:45-52` | IP rate limiter skips Stytch callbacks via `isStytchCallback`. Uses exact path match. | Add wildcard for query params (e.g., `/api/auth/callback?token=...`). |

---

### 4.5 Error Handling & Observability

| Finding | File | Issue |
|---------|------|-------|
| **OBS-1** | `main.go` | Health endpoint `/health` returns `{"status":"ok"}` **without checking DB/Redis**. Load balancer may route to dead instance. |
| **OBS-2** | All | No **metrics endpoint** (`/metrics` for Prometheus). No request latency, error rate, pool usage, queue depth. |
| **OBS-3** | `telemetry/sinks.go:78-82` | `ProcessAll` spawns **unbounded goroutines** per sink per error. Under error storm, goroutine leak. |
| **OBS-4** | `middleware/errors.go:120-130` | `buildTelemetryRequest` captures headers, query, context — may include PII. No sanitization. |
| **OBS-5** | `database/tx.go:88-95` | `DeferRollback` logs `zap.L().Warn` (global logger) instead of passed logger. Inconsistent. |

---

### 4.6 Configuration & Operations

| Finding | File | Issue |
|---------|------|-------|
| **CFG-1** | `config.go:28-38` | `Load()` creates a **new zap logger** just to log `.env` load. Logger not synced properly; may panic on `Sync()`. |
| **CFG-2** | `config.go` | No validation of `DATABASE_URL` format, `REDIS_URL` scheme. Typos cause cryptic startup errors. |
| **CFG-3** | `Dockerfile` | `HEALTHCHECK` uses `wget` to `/health` but `/health` doesn't check deps. |
| **CFG-4** | `main.go:105-110` | Fiber `ReadTimeout: 15s`, `WriteTimeout: 30s`, `IdleTimeout: 60s`. Long timeouts + per-request tx = connection hold. |
| **CFG-5** | `main.go:85-86` | `BodyLimit: 4 MB` but import endpoint overrides with custom `bodySizeLimit` (15 MB). Two limits, confusion. |

---

### 4.7 Code Quality & Maintainability

| Finding | File | Issue |
|---------|------|-------|
| **CODE-1** | `internal/attendance/worker.go` | Raw SQL strings (200+ lines) in handler functions. Hard to test, review, migrate. |
| **CODE-2** | `internal/students/handler.go:260-320` | `List` handler does filtering, pagination, sorting inline. Business logic in handler. |
| **CODE-3** | Multiple | `fx.Invoke` wiring in modules creates **implicit dependencies**; hard to trace startup order. |
| **CODE-4** | `internal/database/asyncq.go` | `asynqRedisOpt` copies Redis options manually. If `go-redis` adds new fields, Asynq client may mismatch. |
| **CODE-5** | `internal/middleware/sessionresolver.go` | `singleflight.Group` keyed by cache key (includes token hash). Memory grows with unique tokens; no eviction. |

---

## 5. Prioritized Improvement Roadmap

### Phase 1 — Immediate (Week 1–2) — **Stability & Safety**

| ID | Task | Effort | Owner |
|----|------|--------|-------|
| P1-1 | Make `MaxConns`, `MinConns`, `RedisPoolSize` configurable via env | 2h | Backend |
| P1-2 | Add DB/Redis health checks to `/health` endpoint | 2h | Backend |
| P1-3 | Remove hardcoded `DATABASE_URL` default with password | 1h | Backend |
| P1-4 | Fix `OBS-3`: bound telemetry goroutines (worker pool or semaphore) | 3h | Backend |
| P1-5 | Add `ReadTimeout`/`WriteTimeout` per-route (shorter for API, longer for imports) | 2h | Backend |
| P1-6 | Sanitize telemetry request (strip auth headers, cookies) | 2h | Backend |

### Phase 2 — Short-term (Month 1) — **Scalability Foundations**

| ID | Task | Effort | Owner |
|----|------|--------|-------|
| P2-1 | **Introduce pgBouncer** (transaction pooling) in front of Postgres. Update pool config to use pgBouncer. | 1w | Platform |
| P2-2 | Split read/write pools: `ReadPool` (larger, no tx) for GET handlers; `WritePool` for mutations. | 3d | Backend |
| P2-3 | Increase Asynq concurrency: attendance=4, imports=8. Add queue priorities. | 1d | Backend |
| P2-4 | Add Prometheus metrics: `http_requests_total`, `db_pool_in_use`, `asynq_queue_depth`, `redis_pool_stats`. | 3d | Backend |
| P2-5 | Implement structured logging correlation (request_id in all log lines) — already via `WithLogger`. Verify. | 1d | Backend |
| P2-6 | Add circuit breaker for Stytch client (e.g., `sony/gobreaker`). | 2d | Backend |

### Phase 3 — Medium-term (Quarter) — **Horizontal Scaling**

| ID | Task | Effort | Owner |
|----|------|--------|-------|
| P3-1 | **Read replicas**: Route analytical queries (summaries, reports) to read replica. | 2w | Platform |
| P3-2 | **Tenant-aware connection pooling**: Per-tenant connection limits to prevent noisy neighbor. | 2w | Backend |
| P3-3 | **Background job partitioning**: Separate Asynq queues per tenant or priority tier. | 1w | Backend |
| P3-4 | **Distributed tracing**: OpenTelemetry + Jaeger/Tempo. Propagate `X-Request-ID` as traceparent. | 2w | Platform |
| P3-5 | **Automated load testing**: k6 scripts for API + background jobs. CI gate on p99 < 500ms. | 1w | QA |

### Phase 4 — Long-term — **Architectural Evolution**

| ID | Task | Effort | Owner |
|----|------|--------|-------|
| P4-1 | **CQRS read models**: Materialized views for dashboards, refreshed by Asynq, queried via read replica. | 1m | Backend |
| P4-2 | **Event-driven architecture**: Domain events (student enrolled, attendance marked) → Kafka/NATS → downstream projections. | 2m | Platform |
| P4-3 | **Multi-region**: Active-passive or active-active with global Redis (Redis Enterprise) and PG logical replication. | 3m | Platform |

---

## 6. Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Pool exhaustion under traffic spike | High | Outage | P1-1, P2-1, P2-2 |
| Migration deadlock on multi-replica deploy | Medium | Deploy failure | Run migrations as separate job (init container), not in app `OnStart`. |
| Stytch outage cascades to auth | Low | Auth down | P2-6 (circuit breaker), cached sessions survive short outage. |
| RLS policy performance regression | Medium | Slow queries | M-1 (index audit), add `pg_stat_statements` monitoring. |
| Asynq Redis saturation | Low | Job processing stalls | P2-3, monitor queue depth, auto-scale workers. |
| PII in telemetry/logs | Medium | Compliance | P1-6 (sanitization), audit log fields. |

---

## 7. Appendix: Key Files Reference

| Area | Files |
|------|-------|
| Entry & DI | `cmd/api/main.go`, `internal/*/module.go` |
| Config | `internal/config/config.go` |
| DB Pool | `internal/database/database.go`, `internal/database/tx.go` |
| Middleware Chain | `internal/middleware/register.go` |
| Tenant Context | `internal/middleware/tenantcontext.go`, `internal/database/tenant.go` |
| Session & Auth | `internal/middleware/sessionresolver.go`, `internal/auth/*` |
| Error Handling | `internal/middleware/errors.go`, `internal/xerrors/xerrors.go` |
| Background Jobs | `internal/attendance/worker.go`, `internal/imports/module.go`, `internal/database/asyncq.go` |
| Telemetry | `internal/telemetry/policy.go`, `internal/telemetry/sinks.go`, `internal/telemetry/module.go` |
| Rate Limiting | `internal/middleware/ratelimiter.go` |
| Docker | `Dockerfile`, `Dockerfile.dev` |

---

## 8. Conclusion

The backend is **well-engineered for correctness and security** (RLS, canonical errors, device-bound sessions, strict DI). The primary scalability limit is the **per-request transaction + fixed pool size**, which caps concurrent authenticated requests at ~25 per instance. Background job processing is also single-threaded for critical summary workloads.

**Top 3 fixes to unblock scaling:**
1. **Configurable pool + pgBouncer** (P1-1, P2-1)
2. **Read/write pool split** (P2-2)
3. **Increase Asynq concurrency + queue priorities** (P2-3)

With these, the service can scale horizontally behind a load balancer to handle **hundreds of req/s** with proper observability and operational tooling.

---
*End of Audit*