# ADR-0022: Observability

## Metadata

| Field | Value |
|-------|-------|
| id | ADR-0022 |
| title | Observability |
| status | proposed |
| date | 2026-11-25 |
| author | backend, platform |
| deciders | backend, platform |
| relates_to | ADR-0005, ADR-0021 |

## Status

proposed

## Context

Somotracker currently has only rate-limit alerts (`backend/monitoring/alerts/ratelimit.yaml`). There is no structured logging, distributed tracing, or metrics standard. The startup integration test verifies DB and Redis connectivity, but there is no observability into application health, request latency, error rates, or backup success/failure.

Without observability, engineers cannot detect or diagnose production issues — slow queries, failed email deliveries, backup failures, or authentication errors.

## Decision

Adopt a **structured logging + metrics + health check** observability stack:

1. **Structured logging**: JSON logs to stdout (captured by the platform). Every log entry includes `timestamp`, `level`, `service`, `request_id`, `user_id` (when authenticated), `organization_id`, and `message`. Log levels: `debug`, `info`, `warn`, `error`. PII (email addresses, phone numbers, OTP codes) is never logged.

2. **Metrics**: Prometheus-compatible metrics exposed via `/metrics` endpoint. Counters and histograms for:
   - HTTP request duration (by endpoint, method, status)
   - HTTP request total (by endpoint, method, status)
   - Database query duration (by query type)
   - Redis operation duration and errors
   - Email delivery success/failure (by provider, message type)
   - Backup success/failure (by type: base, WAL, logical)
   - Active sessions count

3. **Health checks**: `/readyz` endpoint (already verified by startup test) returns 200 when DB ping and Redis ping succeed. `/livez` endpoint returns 200 when the process is alive (no dependency checks).

4. **Distributed tracing**: Request ID propagated through the BFF proxy (Next.js → Go backend) and logged at each service boundary. No full distributed tracing system (OpenTelemetry/Jaeger) in MVP — request ID correlation in logs is sufficient.

5. **Alerting**: Alerts fire on:
   - Backup failure (base, WAL, logical)
   - Database connectivity loss (> 1 minute)
   - Redis connectivity loss (> 1 minute)
   - HTTP 5xx error rate (> 5% over 5 minutes)
   - OTP delivery failure rate (> 10% over 10 minutes)
   - Rate limit fallback to local (Redis down)

## Alternatives Considered

| Alternative | Pros | Cons | Why rejected |
|-------------|------|------|--------------|
| No observability | Simplest | Cannot detect or diagnose production issues | Rejected — operational blindness |
| Full distributed tracing (OpenTelemetry) | End-to-end request visibility | High complexity, additional infrastructure | Rejected — request ID correlation sufficient for MVP |
| Structured logging + metrics + health checks (chosen) | Actionable signals, low complexity | No end-to-end trace visualization | Accepted — MVP scope |

## Consequences

### Positive

- Engineers can detect and diagnose production issues via logs and metrics.
- Health checks support container orchestration (liveness/readiness probes).
- Alerting provides early warning of backup failures, DB/Redis outages, and email delivery issues.
- Request ID correlation enables tracing requests across the BFF proxy boundary.

### Negative

- Metrics and logs require storage and retention management.
- Alert thresholds need tuning to avoid alert fatigue.
- No distributed trace visualization (can be added later).

### Neutral / Risks

- PII must be scrubbed from logs — review required before production.
- Metrics endpoint must be protected (internal network only or authenticated).

## Implementation Notes

1. **Logging**: Use Go's `log/slog` with JSON handler. Middleware injects `request_id` into logger context.
2. **Metrics**: Use `prometheus_client_golang` with custom collectors for DB, Redis, email, and backup.
3. **Health checks**: `/readyz` and `/livez` handlers in `backend/internal/app/`.
4. **Alerting**: Alertmanager rules in `backend/monitoring/alerts/` (extend `ratelimit.yaml` with new alerts).

## Links

- Rate limiting: `docs/adr/0005-rate-limiting-strategy.md`
- Backup & DR: `docs/adr/0021-backup-disaster-recovery.md`
- Secrets rotation: `docs/adr/0023-secrets-rotation.md` (planned)
