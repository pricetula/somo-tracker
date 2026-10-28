# Open Risks – Authentication Design

This doc captures residual risks and open questions from the revised Stytch B2B Discovery + TanStack Start authentication design. Use it as input for deeper analysis.

## 1. Onboarding credential gap
**Status:** Resolved with short-lived onboarding ticket
**Decision:** At the end of Discovery authenticate, Go issues a host-only cookie `__Host-onboarding` on `api.domain.com` with HttpOnly, Secure, Path=/, TTL ≤ IST TTL. Server stores ticket hash + encrypted IST against pending record id, single-use, deleted on use. Ticket is accepted only on tenant-name submit and pending-record read, never in protected session middleware. Expired ticket returns `onboarding_expired` → frontend redirects to `/login` for fresh magic link.
**Enforcement:** Every route has explicit auth: 'none' | 'ticket' | 'session'. CI test fails if a route lacks it.
**Risk:** Link opened in different browser/device fails because verifier cookie is host-only. UX message and resend needed.

## 2. Stytch PKCE for login-CSRF
**Status:** Decided – enable PKCE
**Decision:** Stytch Discovery supports PKCE. Go generates `pkce_code_verifier`, stores it in a short-lived HttpOnly cookie, sends `pkce_code_challenge` to Stytch on magic-link send. On callback, Go reads verifier cookie and passes it to Discovery authenticate. Cookie TTL matches link expiry (60 min default). Device mismatch will fail – provide clear UX and resend.
**Risk:** Cross-device link opening fails. Acceptable trade-off for CSRF protection.

## 3. Exact Stytch Go SDK Discovery API surface
**Status:** Open
**Risk:** Design references `POST /api/v1/auth/stytch/discovery/authenticate`, IST exchange, organization creation via Discovery. Method names and payload shapes depend on pinned Stytch Go SDK version.
**Decision needed:** Pin SDK version and verify exact method names, request/response fields, and error codes. Update OpenAPI spec accordingly.

## 4. Parent-domain cookie exposure
**Status:** Accepted with controls
**Risk:** `Domain=domain.com` cookie is sent to all sibling subdomains. Sibling subdomain can also set a cookie with same name for Domain=domain.com. __Secure- prefix does not prevent this.
**Mitigation:** Per-environment cookie name suffix, enforce HTTPS, restrict subdomain creation, reject requests with duplicate session cookies, rotate token at login. Recommended long-term: same-origin API under `app.domain.com/api/*` to enable `__Host-` cookie and remove CORS.

## 5. CSRF Origin missing behavior
**Status:** Decided
**Decision:** For mutations, validate Origin by exact string equality against allowed origin. Reject `Origin: null`. If Origin is missing, allow only when `Sec-Fetch-Site` is `same-origin`. `Sec-Fetch-Site: same-site` is untrusted because sibling subdomains appear same-site. Harden later with double-submit token.

## 6. Session lifetime policy
**Status:** Decided: application-managed
**Risk:** No Stytch session revalidation. If Stytch session is revoked externally, application session remains valid until expiry/revocation.
**Mitigation:** Documented as policy choice. Can add periodic revalidation later with `last_checked_at`.

## 7. Redis miss fallback
**Status:** Decided: fallback to PostgreSQL with safeguards
**Decision:** Keep cache TTL 30-60 seconds, delete cache key on logout/revoke, never fail open on Redis errors, limit DB concurrency with circuit breaker, briefly cache unknown tokens to avoid Postgres hammering.
**Risk:** Cache stampede under load.
**Mitigation:** Monitor hit rate and latency.

## 8. Email change and account linking
**Status:** Decided
**Decision:** Email change requires fresh Stytch verification + admin approval. No automatic merge of pending records. For tenant with only one admin, define platform-support escalation path.

## 9. Pending onboarding cleanup
**Status:** Decided
**Decision:** Soft expire after 90 days inactivity, purge after 180 days.
**Risk:** Users who pause >90 days lose progress. Ensure UX communicates expiry.

## 10. `?redirect=` open redirect
**Status:** Decided
**Decision:** Allow only same-origin relative paths via allowlist.
**Risk:** Implementation must strictly validate. Avoid using user-supplied absolute URLs.

## 11. Middleware allowlist fragility
**Status:** Accepted
**Risk:** `Accept: text/html` and `Sec-Fetch-Dest` checks are defense-in-depth, not primary gate. Misconfiguration could redirect server-function calls.
**Mitigation:** Keep allowlist as primary control, test server-function paths explicitly.

## 12. Data fetching client-only limitation
**Status:** Decided
**Risk:** No SSR for protected data. SEO impact minimal for admin app, but page load will show skeleton then data.
**Mitigation:** Accept for now. Revisit if SSR needed.

## 13. Logout preloading and CSRF
**Status:** Decided – JSON POST via button
**Decision:** Logout is `POST /api/v1/auth/logout` via JS `fetch` with JSON body and `credentials: 'include'`. Use a `<button>` with `onClick`, not a link. This avoids router preloading and avoids form-encoded body rejection. Origin check applies to logout too.
**Risk:** Ensure button is not accidentally triggered by prefetch.

## 14. Token entropy and storage
**Status:** Decided
**Decision:** ≥256 bits entropy, SHA-256 hash stored, rotate at login.
**Risk:** Implementation must ensure cryptographically secure RNG and constant-time compare.

## 15. Audit and logging
**Status:** Decided
**Risk:** Magic-link token in query string reaches CDN, reverse proxy, TanStack server access logs, error tracker URLs/breadcrumbs before Go redaction.
**Mitigation:** Redact tokens at CDN/proxy layer, scrub Cookie/Set-Cookie/Authorization headers in Fiber logger and tracing. PKCE reduces blast radius of leaked token. No secrets in logs at any layer.

## Next steps for deeper analysis
- Validate Stytch PKCE and exact SDK methods.
- Threat model parent-domain cookie with current subdomain inventory.
- Load test session validation path with Redis fallback.
- Define monitoring/alerting for 401 spikes, stale cookie redirects, and provisioning failures.
