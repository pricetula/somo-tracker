# SPEC-015: Rate Limiting API Contract (OpenAPI 3.1)

Endpoint: Standard headers and error responses across all `/api/v1/*` routes

## Metadata

| Field | Value |
|-------|-------|
| id | SPEC-015 |
| title | Rate Limiting API Contract — Headers & Error Responses |
| status | draft |
| depends_on | docs/adr/0002-rate-limiting-strategy.md, specs/010-ratelimit-phase0-trust-boundary.md, specs/011-ratelimit-phase1-core-limiter.md, specs/012-ratelimit-phase2-otp-hardening.md, specs/013-ratelimit-phase3-bot-defense.md, specs/014-ratelimit-phase4-global-middleware-observability.md |
| api_endpoint | ALL /api/v1/* |
| owner | fullstack |

## Summary

Defines the **standardized rate limit headers** (IETF `RateLimit-*` + legacy `X-RateLimit-*`), **RFC 9457 problem details** for `429 Too Many Requests` and `428 Precondition Required`, and **challenge response formats** for bot defense step-up. All endpoints under `/api/v1/*` must conform.

---

## Rate Limit Headers

### Standard Headers (IETF Draft / RFC 6585)

| Header | Format | Example | Description |
|--------|--------|---------|-------------|
| `RateLimit-Limit` | integer | `300` | Maximum requests allowed in the window |
| `RateLimit-Remaining` | integer | `287` | Requests remaining in current window |
| `RateLimit-Reset` | integer (seconds) | `45` | Seconds until window resets |

### Legacy Headers (Compatibility)

| Header | Format | Example | Description |
|--------|--------|---------|-------------|
| `X-RateLimit-Limit` | integer | `300` | Same as `RateLimit-Limit` |
| `X-RateLimit-Remaining` | integer | `287` | Same as `RateLimit-Remaining` |
| `X-RateLimit-Reset` | integer (unix timestamp) | `1699900000` | Unix timestamp when window resets |

### Retry-After (RFC 9110)

| Header | Format | Example | When Present |
|--------|--------|---------|--------------|
| `Retry-After` | integer (seconds) | `42` | **Only on 429 responses** |

### Header Presence Rules

| Response | Headers Present |
|----------|-----------------|
| `2xx` / `4xx` (non-429) | `RateLimit-Limit`, `RateLimit-Remaining`, `RateLimit-Reset`, `X-RateLimit-*` |
| `429 Too Many Requests` | All above + `Retry-After` |
| `428 Precondition Required` | All above (no `Retry-After`) |
| `503 Service Unavailable` | No rate limit headers (fail-closed) |

---

## Error Response Formats

### 429 Too Many Requests (RFC 9457)

**Content-Type**: `application/problem+json`

```json
{
  "type": "https://api.example.com/problems/rate-limited",
  "title": "Too many requests",
  "status": 429,
  "code": "rate_limited",
  "retry_after_seconds": 42
}
```

**Fields:**
- `type`: URI identifying the problem type (stable)
- `title`: Human-readable summary
- `status`: HTTP status code (429)
- `code`: Machine-readable error code (`rate_limited`)
- `retry_after_seconds`: Seconds until client may retry (matches `Retry-After` header)

> **Critical**: Response **never discloses** which key (`email`, `IP`, `challengeID`, `userID`, `global`) triggered the limit. This prevents account enumeration.

### 428 Precondition Required — Step-Up Challenge (RFC 9457)

**Content-Type**: `application/problem+json`

```json
{
  "type": "https://api.example.com/problems/challenge-required",
  "title": "Additional verification required",
  "status": 428,
  "code": "challenge_required",
  "challenge": {
    "type": "pow",
    "difficulty": 6,
    "seed": "base64_encoded_seed",
    "expires_in_seconds": 120
  }
}
```

**Challenge Types:**

| Type | Fields | Description |
|------|--------|-------------|
| `pow` | `difficulty`, `seed`, `expires_in_seconds` | Proof-of-Work (HMAC-signed) |
| `turnstile` | `sitekey`, `action` | Cloudflare Turnstile widget |
| `webauthn` | `challenge`, `rp_id`, `allow_credentials` | WebAuthn/Passkey assertion |

### 403 Forbidden — BFF Auth Failure (RFC 9457)

```json
{
  "type": "https://api.example.com/problems/forbidden",
  "title": "Forbidden",
  "status": 403,
  "code": "forbidden",
  "detail": "Invalid or missing BFF service credential"
}
```

### 503 Service Unavailable — Fail-Closed (RFC 9457)

```json
{
  "type": "https://api.example.com/problems/service-unavailable",
  "title": "Service temporarily unavailable",
  "status": 503,
  "code": "service_unavailable",
  "retry_after_seconds": 30
}
```

---

## Bot Defense Endpoints

### `POST /api/v1/auth/challenge` — Issue PoW Challenge

**Request:**
```json
{
  "action": "otp_request",
  "email_hash": "a1b2c3d4...",  // HMAC-SHA256 hex
  "client_subnet": "1.2.3.0/24"
}
```

**Response (200):**
```json
{
  "challenge_id": "550e8400-e29b-41d4-a716-446655440000",
  "difficulty": 4,
  "expires_at": "2026-11-24T12:02:00Z",
  "seed": "base64_encoded_random_bytes"
}
```

**Error Responses:**
- `400 INVALID_REQUEST` — Missing/invalid fields
- `429 RATE_LIMITED` — Challenge issuance rate limited
- `503 SERVICE_UNAVAILABLE` — Redis unavailable (fail-closed)

### `POST /api/v1/auth/otp/request` — Extended Request

**Request:**
```json
{
  "email": "user@example.com",
  "turnstile_token": "cf_token_from_widget",
  "pow_solution": {
    "challenge_id": "550e8400-e29b-41d4-a716-446655440000",
    "nonce": "deadbeef..."
  },
  "honeypot_field": ""
}
```

**Headers from BFF (auto-injected):**
- `X-BFF-Form-Dwell-Ms: 1250`
- `X-Client-IP: 1.2.3.4`
- `X-BFF-Service-Secret: <secret>`

**Success Response (200):**
```json
{
  "ok": true,
  "message": "If an account exists for this email, a code has been sent.",
  "challenge_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Rate Limit Headers:** All standard headers present.

---

## Header Examples by Tier

### Global API (Authenticated User)

```http
HTTP/1.1 200 OK
RateLimit-Limit: 300
RateLimit-Remaining: 299
RateLimit-Reset: 55
X-RateLimit-Limit: 300
X-RateLimit-Remaining: 299
X-RateLimit-Reset: 1699900055
Content-Type: application/json
```

### OTP Request (Email-Keyed)

```http
HTTP/1.1 200 OK
RateLimit-Limit: 3
RateLimit-Remaining: 2
RateLimit-Reset: 840
X-RateLimit-Limit: 3
X-RateLimit-Remaining: 2
X-RateLimit-Reset: 1699900840
Content-Type: application/json
```

### OTP Request — 429 Response

```http
HTTP/1.1 429 Too Many Requests
RateLimit-Limit: 3
RateLimit-Remaining: 0
RateLimit-Reset: 840
X-RateLimit-Limit: 3
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1699900840
Retry-After: 840
Content-Type: application/problem+json

{
  "type": "https://api.example.com/problems/rate-limited",
  "title": "Too many requests",
  "status": 429,
  "code": "rate_limited",
  "retry_after_seconds": 840
}
```

### Onboard — 428 Step-Up Challenge

```http
HTTP/1.1 428 Precondition Required
RateLimit-Limit: 5
RateLimit-Remaining: 4
RateLimit-Reset: 45
X-RateLimit-Limit: 5
X-RateLimit-Remaining: 4
X-RateLimit-Reset: 1699900045
Content-Type: application/problem+json

{
  "type": "https://api.example.com/problems/challenge-required",
  "title": "Additional verification required",
  "status": 428,
  "code": "challenge_required",
  "challenge": {
    "type": "pow",
    "difficulty": 6,
    "seed": "base64_encoded_seed",
    "expires_in_seconds": 120
  }
}
```

---

## Client Integration Guide

### Backoff Algorithm (Recommended)

```javascript
async function apiRequest(url, options = {}) {
  const response = await fetch(url, options);
  
  if (response.status === 429) {
    const retryAfter = response.headers.get('Retry-After');
    const seconds = retryAfter ? parseInt(retryAfter, 10) : 60;
    
    // Exponential backoff with jitter
    const delay = seconds * 1000 + Math.random() * 1000;
    await new Promise(r => setTimeout(r, delay));
    
    return apiRequest(url, options); // Retry once
  }
  
  if (response.status === 428) {
    const problem = await response.json();
    if (problem.code === 'challenge_required') {
      // Solve challenge based on type
      await solveChallenge(problem.challenge);
      return apiRequest(url, options); // Retry with challenge solution
    }
  }
  
  return response;
}
```

### Reading Rate Limit Headers

```javascript
function getRateLimitInfo(response) {
  return {
    limit: parseInt(response.headers.get('RateLimit-Limit') || '0', 10),
    remaining: parseInt(response.headers.get('RateLimit-Remaining') || '0', 10),
    resetSeconds: parseInt(response.headers.get('RateLimit-Reset') || '0', 10),
    resetTimestamp: parseInt(response.headers.get('X-RateLimit-Reset') || '0', 10) * 1000,
  };
}

// UI: Show warning when remaining < 10% of limit
const info = getRateLimitInfo(response);
if (info.remaining / info.limit < 0.1) {
  showRateLimitWarning(info.resetSeconds);
}
```

---

## OpenAPI 3.1 Schema (Excerpt)

```yaml
components:
  headers:
    RateLimit-Limit:
      schema:
        type: integer
        minimum: 0
      description: Maximum requests allowed in the current window
    RateLimit-Remaining:
      schema:
        type: integer
        minimum: 0
      description: Requests remaining in the current window
    RateLimit-Reset:
      schema:
        type: integer
        minimum: 0
      description: Seconds until the rate limit window resets
    X-RateLimit-Limit:
      schema:
        type: integer
        minimum: 0
      description: Legacy header - maximum requests allowed
    X-RateLimit-Remaining:
      schema:
        type: integer
        minimum: 0
      description: Legacy header - requests remaining
    X-RateLimit-Reset:
      schema:
        type: integer
        minimum: 0
      description: Legacy header - Unix timestamp when window resets
    Retry-After:
      schema:
        type: integer
        minimum: 0
      description: Seconds to wait before retrying (429 only)

  schemas:
    ProblemDetails:
      type: object
      required: [type, title, status, code]
      properties:
        type:
          type: string
          format: uri
        title:
          type: string
        status:
          type: integer
        code:
          type: string
        detail:
          type: string
        retry_after_seconds:
          type: integer
        challenge:
          $ref: '#/components/schemas/Challenge'

    Challenge:
      type: object
      required: [type]
      properties:
        type:
          type: string
          enum: [pow, turnstile, webauthn]
        difficulty:
          type: integer
        seed:
          type: string
        expires_in_seconds:
          type: integer
        sitekey:
          type: string
        action:
          type: string
```

---

## Contract Tests Required

| Test Type | What to Verify |
|-----------|----------------|
| Contract | All `/api/v1/*` 2xx/4xx responses include `RateLimit-Limit`, `RateLimit-Remaining`, `RateLimit-Reset` |
| Contract | All `/api/v1/*` 2xx/4xx responses include legacy `X-RateLimit-*` headers |
| Contract | `429` responses include `Retry-After` header (seconds) |
| Contract | `429` response body matches RFC 9457 `ProblemDetails` schema (`code: rate_limited`) |
| Contract | `428` response body matches RFC 9457 with `challenge` object (`code: challenge_required`) |
| Contract | `403`/`400`/`503` responses match RFC 9457 schemas |
| Contract | No rate limit headers on `503` fail-closed responses |
| Contract | `POST /auth/challenge` 200 response matches `Challenge` schema |
| Contract | OpenAPI 3.1 schema validates against Spectral/Redoc |
| Security | `429` response never discloses limiting key (`email`, `ip`, `challengeID`, `userID`, `global`) |
| Security | `401` responses identical for all failure types (no enumeration) |
| Security | Error responses never leak internal details (stack traces, Redis keys, etc.) |

## Links

- ADR: `docs/adr/0002-rate-limiting-strategy.md` §7
- RFC 6585: Additional HTTP Status Codes
- RFC 9110: HTTP Semantics (Retry-After)
- RFC 9457: Problem Details for HTTP APIs
- Phase 4: `specs/014-ratelimit-phase4-global-middleware-observability.md`