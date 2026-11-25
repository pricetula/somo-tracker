# ADR-0024: API Versioning

## Metadata

| Field | Value |
|-------|-------|
| id | ADR-0024 |
| title | API Versioning |
| status | proposed |
| date | 2026-11-25 |
| author | backend, frontend |
| deciders | backend, frontend |
| relates_to | ADR-0006 |

## Status

proposed

## Context

Somotracker's Go backend exposes REST API under `/api/v1/*` (e.g. `/api/v1/auth/otp/request`). The frontend is a Next.js BFF proxy (ADR-0006) that proxies `/api/auth/*` to `/api/v1/auth/*`. There is no documented API versioning, deprecation, or breaking-change policy. As the education configuration domain expands (ADR-0012 through ADR-0018), new endpoints will be added.

Without a versioning strategy, breaking changes to the API could break the frontend or any future clients.

## Decision

Adopt a **URL-path versioning with backward-compatible changes** model:

1. **Version in URL path**: API version is embedded in the URL path (`/api/v1/*`). The current version is `v1`.

2. **Backward-compatible changes only within a version**: Adding new endpoints, new optional request fields, new optional response fields, and new enum values are allowed within `v1` without a version bump. These changes do not break existing clients.

3. **Breaking changes require a new version**: Removing endpoints, removing request/response fields, changing field types, changing enum values, or changing URL paths require a new version (`v2`). The old version is maintained for a deprecation period.

4. **Deprecation policy**: When a new version is released, the previous version is deprecated for a minimum of 6 months before removal. Deprecation is communicated via API response headers (`Deprecation` header with sunset date) and documentation.

5. **BFF proxy versioning**: The Next.js BFF proxy (ADR-0006) forwards requests to the backend version specified in the route. The frontend controls which backend version it calls — frontend and backend can be versioned independently.

6. **No header-based versioning**: Version is not determined by `Accept` or `X-API-Version` headers — URL path is the single source of truth.

## Alternatives Considered

| Alternative | Pros | Cons | Why rejected |
|-------------|------|------|--------------|
| No versioning | Simplest | Breaking changes break clients without notice | Rejected — frontend and future clients need stability |
| Header-based versioning (`Accept: application/vnd.somotracker.v1+json`) | Clean URLs, multiple versions on same path | Complex to implement, hard to debug, header can be lost | Rejected — URL path is explicit and cacheable |
| URL-path versioning (chosen) | Simple, explicit, cacheable, easy to debug | URLs change when version bumps | Accepted — clarity outweighs URL stability |

## Consequences

### Positive

- Clients know exactly which API version they are calling.
- Backward-compatible changes can be shipped without version bumps.
- Breaking changes are intentional and communicated.
- BFF proxy decouples frontend versioning from backend versioning.

### Negative

- New URLs for each major version (clients must update to call `v2`).
- Multiple versions must be maintained simultaneously during deprecation period.

### Neutral / Risks

- Backend must support multiple versions concurrently during deprecation (code complexity).
- Frontend must update proxy routes when adopting a new backend version.

## Implementation Notes

1. **Router**: Go Fiber router groups by version (`/api/v1/*`). Middleware can inject `Deprecation` header for deprecated versions.
2. **BFF proxy**: Next.js API routes under `/api/auth/*` proxy to `/api/v1/auth/*`. When backend moves to `v2`, frontend proxy routes can be updated independently.
3. **Documentation**: API version and changelog maintained in `docs/` (Postman stubs, spec docs).

## Links

- BFF proxy: `docs/adr/0006-bff-proxy-architecture.md`
