# ADR-0006: BFF Proxy Architecture

## Metadata

| Field | Value |
|-------|-------|
| id | ADR-0006 |
| title | BFF Proxy Architecture |
| status | proposed |
| date | 2026-11-25 |
| author | engineering |
| deciders | frontend, backend |
| relates_to | ADR-0001, ADR-0003 |

## Status

proposed

## Context

The frontend is a Next.js application that acts as a Backend-for-Frontend (BFF) proxy to the Go backend API. The frontend does not call the Go backend directly from the browser; instead, it calls Next.js API routes under `/api/auth/*` which proxy to `/api/v1/auth/*` on the Go backend.

Key forces:
- Session cookies (`__Host-session`, `needs_onboarding`) must be set by the Go backend and flow through Next.js to the browser.
- The frontend proxy must forward cookies (`credentials: include`) so the backend can authenticate requests.
- Onboarding status (`needs_onboarding`) drives client-side routing (redirect to `/onboarding` vs `/dashboard`).
- The BFF layer can add frontend-specific concerns (API response shaping, cookie forwarding, route guards).
- Docs reference `docs/bff-proxy.md` for the proxy implementation.

Key decisions needed:
- How cookies flow through the proxy.
- Which routes are proxied vs handled directly by Next.js.
- How the frontend determines onboarding state without a separate backend call.
- Error propagation from backend to frontend.

## Decision

Adopt a Next.js BFF proxy with cookie passthrough and session-driven client routing:

1. **Proxy routes**: Next.js API routes under `/api/auth/*` (otp/request, otp/verify, onboard, session, logout) proxy to the Go backend at `/api/v1/auth/*`. Other API routes may be added as needed.
2. **Cookie forwarding**: Frontend fetch calls use `credentials: 'include'` so browser cookies (session, needs_onboarding) are sent to the Next.js proxy, which forwards them to the Go backend. Backend `Set-Cookie` headers flow back through Next.js to the browser.
3. **Cookie domain**: Backend sets `__Host-session` (host-only, no domain) which works through the proxy since the cookie is set on the response that flows from backend → Next.js → browser.
4. **Onboarding routing**: The `/session` endpoint returns `needs_onboarding`; the `needs_onboarding` cookie is also readable by the frontend proxy. On every navigation, the proxy/middleware checks:
   - No session → redirect `/login`
   - Session + `needs_onboarding=true` → redirect `/onboarding`
   - Session + `needs_onboarding=false` → allow access
5. **Error handling**: Backend errors return standardized JSON (`{ code, message }`); the frontend proxy surfaces `message` to the user. HTTP status codes are preserved.
6. **Dynamic routes**: Proxy route handlers use `export const dynamic = 'force-dynamic'` to avoid static caching of auth responses.
7. **Base path**: Frontend calls relative `/api/auth/*`; the proxy rewrites to the configured backend base URL.

## Alternatives Considered

| Alternative | Pros | Cons | Why rejected |
|-------------|------|------|--------------|
| Frontend calls Go backend directly (CORS) | Simpler architecture, no proxy layer | CORS configuration, cookie cross-domain complexity, CSRF exposure | Proxy avoids CORS and keeps cookies same-site |
| Next.js server actions | No explicit API routes | Less suitable for cookie-based auth flows, harder to type | API routes are clearer for auth endpoints |
| GraphQL BFF | Strongly typed, flexible queries | Over-engineering for auth, adds GraphQL layer | REST API routes are sufficient |
| Edge middleware for auth checks | Fast, runs at edge | Limited access to request body/cookies in some cases | Next.js API routes + middleware hybrid is more flexible |

## Consequences

### Positive

- Cookies stay same-origin; no CORS configuration needed.
- `__Host-` cookie prefix works naturally through the proxy response flow.
- Frontend controls routing based on `needs_onboarding` without extra backend calls per navigation.
- Central place to add frontend-specific concerns (response shaping, logging, error UI).
- TypeScript API client (`frontend/src/lib/api-client.ts`) centralizes fetch logic.

### Negative

- Extra network hop (browser → Next.js → Go) adds latency to auth requests.
- Next.js must be running for the frontend to function; cannot call Go backend directly during development without the proxy.
- Proxy routes must be carefully secured (no open proxy to arbitrary backend paths).

### Neutral / Risks

- `docs/bff-proxy.md` exists but should be kept in sync with this ADR and the actual proxy implementation (`frontend/src/proxy.ts`, `frontend/src/app/api/auth/onboard/route.ts`).
- Cookie forwarding must be verified across dev (HTTP) and prod (HTTPS `__Host-`) environments.
- If the backend sets cookies on a different path/domain than the proxy expects, cookies may not be forwarded back on subsequent requests.

## Links

- BFF proxy doc: `docs/bff-proxy.md`
- Proxy implementation: `frontend/src/proxy.ts`
- Onboard route: `frontend/src/app/api/auth/onboard/route.ts`
- API client: `frontend/src/lib/api-client.ts`
- Login form: `frontend/src/components/login-form.tsx`
- Onboarding page: `frontend/src/app/onboarding/page.tsx`
- Auth ADR: `docs/adr/0001-authentication-architecture.md`
- Session ADR: `docs/adr/0003-session-management.md`
