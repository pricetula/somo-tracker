---
id: ARCH-ERROR-HANDLING
title: Error Handling, Request ID & Logging Contract
status: draft
depends_on: []
touches: [backend, frontend]
---

# Error Handling, Request ID & Logging Contract

*Draft v2. Supersedes draft v1 of ARCH-ERROR-HANDLING. Covers backend (Go, Fiber v3) and frontend (TanStack Start).*

## 1. Purpose and instructions for the implementing agent

This document is the normative contract for how errors are shaped, identified, logged and consumed across the API and the web app. MUST, SHOULD and MAY follow RFC 2119. It is meant to be turned into spec-driven development documents, one per phase in section 13.

Rules for the agent:

- Treat sections 4 to 10 as requirements. Code samples are reference implementations, not paste-ready code. Check `go.mod` and `package.json` for exact versions and verify every Fiber v3, validator v10 and TanStack Start API against the installed version before using it. Anything marked VERIFY could not be confirmed by the author.
- Read the existing code first (`platform/httpserver/server.go`, the `apperr` package, current handlers, `src/lib/api/client.ts`, `src/lib/errors.ts`) so each spec states what exists and what changes.
- Every generated spec MUST include: goal, files to create or modify, interface signatures, acceptance criteria traceable to section numbers here, tests, and rollback notes.
- Do not change the contract (sections 4 and 5) inside a downstream spec. Propose changes by editing this document.
- File paths are suggestions. Follow repo conventions where they differ.

## 2. Context

| Area | Facts |
| --- | --- |
| Backend | Go, Fiber v3, `log/slog`, go-playground/validator v10, internal `apperr` package, cookie-session auth, multi-tenant (cross-tenant access returns 404) |
| Frontend | TanStack Start (SSR plus client), TanStack Router, TanStack Query |
| Error reporting | slog only. Sentry is deliberately deferred (section 14) |
| Current state | Backend returns `{code, message}` such as `INVALID_EMAIL`, with no `errors` map and no `request_id`. Frontend `apiFetch` throws an object literal and `getErrorMessage` flattens raw field names |

Out of scope: i18n implementation, rate limiter implementation, Sentry/OpenTelemetry, metrics.

## 3. Decisions (changes from draft v1)

| # | Decision | Rationale |
| --- | --- | --- |
| D1 | Request ID is `req_` + 22 chars (16 random bytes, base64 RawURL, 128 bits); 26 chars total | v1 said 16 chars, contradicting 128-bit entropy |
| D2 | Field errors are objects `{code, message}`, not bare strings | Frontend can localize and branch without parsing text; painful to retrofit |
| D3 | `code` is either a generic HTTP-level code or a registered domain code (for example `EMAIL_TAKEN`). One registry maps each code to one status | Avoids losing information like `INVALID_EMAIL` becoming `VALIDATION_FAILED` |
| D4 | 400 for unparseable requests, 422 for parseable requests with invalid values (including JSON type mismatches) | Removes ambiguity (5.3) |
| D5 | 5xx messages are always generic; details go to logs only | Prevents leaking internals |
| D6 | Log level follows status class (8.3) | Avoids noisy alerting |
| D7 | Request ID is generated at the first hop. Start generates or forwards one for server-side calls; the Go API accepts a well-formed inbound ID or generates its own | One ID per page load across both tiers |
| D8 | Response body is a typed struct, never `fiber.Map` | Prevents shape drift |
| D9 | Contract covers 4xx and 5xx only | 3xx and 304 are not errors |
| D10 | slog only for now; a single hook point is reserved for later reporting | Simplicity |

## 4. Error response contract

Every 4xx and 5xx response from `/api/v1/*`, including router-level 404/405, body-limit errors, recovered panics and timeouts, MUST be `Content-Type: application/json` with exactly this shape:

```json
{
  "code": "VALIDATION_FAILED",
  "message": "Request validation failed",
  "errors": {
    "email": [{ "code": "invalid_format", "message": "must be a valid email address" }],
    "items[0].name": [{ "code": "required", "message": "is required" }]
  },
  "request_id": "req_k3Jd9sLmP0qX7vTnA2bC8w"
}
```

| Field | Type | Rules |
| --- | --- | --- |
| `code` | string | SCREAMING\_SNAKE\_CASE, stable, from the registry (section 5). Clients branch on this, never on `message` |
| `message` | string | Human-readable English, safe to display. Always generic for 5xx |
| `errors` | object | Map of field key to array of `{code, message}`. Always present, `{}` when none, never null |
| `request_id` | string | Identical to the `X-Request-ID` response header |

Rules:

- The server MUST NOT put stack traces, SQL, file paths, hostnames or user-supplied values in any message.
- Error responses SHOULD set `Cache-Control: no-store`. 429 MUST and 503 MAY set `Retry-After` (seconds).
- Success responses SHOULD also carry `X-Request-ID`.
- Errors after a stream has started (SSE, chunked) cannot change the status. The server logs at Error level and, for SSE, sends one terminal `event: error` whose data is this JSON body, then closes.

## 5. Error codes

### 5.1 Generic codes

| Code | Status | Use |
| --- | --- | --- |
| `BAD_REQUEST` | 400 | Malformed JSON or otherwise unparseable request |
| `UNAUTHENTICATED` | 401 | Missing, invalid or expired session |
| `FORBIDDEN` | 403 | Authenticated but not permitted |
| `CSRF_FAILED` | 403 | CSRF token/origin check failed. Distinct so the client can refresh instead of showing a permission error |
| `NOT_FOUND` | 404 | Missing resource, outside tenant scope, or unmatched route |
| `METHOD_NOT_ALLOWED` | 405 | Route exists, method does not |
| `REQUEST_TIMEOUT` | 408 | Client too slow sending the request |
| `CONFLICT` | 409 | Idempotency conflict, duplicate resource |
| `PAYLOAD_TOO_LARGE` | 413 | Body over limit |
| `UNSUPPORTED_MEDIA_TYPE` | 415 | Content-Type not accepted |
| `VALIDATION_FAILED` | 422 | Parseable request with invalid values |
| `RATE_LIMITED` | 429 | Must include `Retry-After` |
| `INTERNAL_ERROR` | 500 | Unexpected failure |
| `BAD_GATEWAY` | 502 | Invalid upstream response |
| `SERVICE_UNAVAILABLE` | 503 | Overloaded or shutting down |
| `GATEWAY_TIMEOUT` | 504 | Upstream timed out |

`statusToCode` fallback: an unlisted 4xx maps to `CLIENT_ERROR`, an unlisted 5xx to `INTERNAL_ERROR`; the original status is preserved.

### 5.2 Domain codes

Domain codes (for example `EMAIL_TAKEN`, `INVITE_EXPIRED`) are registered in one file (`apperr/codes.go`) via `Register(code, status)`. A code has exactly one status and one meaning forever. Adding is non-breaking; renaming or removing is breaking. A test MUST fail if a code is used but unregistered. The frontend keeps a matching union type (generated from the registry or an OpenAPI schema where possible).

### 5.3 400 vs 422

- 400: body is not valid JSON, empty when required, malformed multipart.
- 422: JSON parses but a field is missing, has the wrong type (`"age": "abc"` gives field code `invalid_type`) or violates rules. Failing query and path parameter validation is also 422.
- A well-formed ID that does not exist is 404, not 400.

### 5.4 Field keys and field codes

- Keys use JSON names, never Go field names (register a validator `TagNameFunc`).
- Nested: `address.city`. Arrays: `items[0].name`.
- Non-body sources are prefixed: `query.page`, `path.id`, `header.x-foo`.
- Errors not tied to a field use key `_form`.

| validator tag | field code |
| --- | --- |
| required | required |
| email, url, uuid, datetime, e164 | invalid\_format |
| min / max on strings | too\_short / too\_long |
| min, max, gte, lte, gt, lt on numbers | out\_of\_range |
| min / max on slices | too\_few / too\_many |
| oneof | invalid\_value |
| JSON type mismatch | invalid\_type |
| anything else | invalid |

## 6. Request ID

- Generated by the first middleware, before logging, recover and routing, so router-level errors and panics carry an ID.
- An inbound `X-Request-ID` is accepted only if it matches `^[A-Za-z0-9_-]{8,64}$`; otherwise a new one is generated. This prevents log injection and bloat.
- Stored in: `c.Locals("request_id")`, `context.Context` under a private typed key, the request-scoped logger, and the response header.
- Service code reads it with `RequestIDFrom(ctx)` and MUST NOT depend on Fiber locals.
- Outbound HTTP calls and enqueued jobs SHOULD propagate it.

```go
type ctxKey int

const (
  requestIDKey ctxKey = iota
  loggerKey
)

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{8,64}$`)

func newRequestID() string {
  b := make([]byte, 16)
  if _, err := rand.Read(b); err != nil { // crypto/rand
    return "req_unknown"
  }
  return "req_" + base64.RawURLEncoding.EncodeToString(b)
}

func RequestID() fiber.Handler {
  return func(c fiber.Ctx) error {
    id := c.Get("X-Request-ID")
    if !requestIDPattern.MatchString(id) {
      id = newRequestID()
    }
    logger := slog.Default().With("request_id", id)
    ctx := context.WithValue(c.Context(), requestIDKey, id)
    ctx = context.WithValue(ctx, loggerKey, logger)
    c.SetContext(ctx) // VERIFY: Fiber v3 Context()/SetContext semantics
    c.Locals("request_id", id)
    c.Set("X-Request-ID", id)
    return c.Next()
  }
}

func RequestIDFrom(ctx context.Context) string {
  if id, ok := ctx.Value(requestIDKey).(string); ok {
    return id
  }
  return "req_unknown"
}

func LoggerFrom(ctx context.Context) *slog.Logger {
  if l, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
    return l
  }
  return slog.Default()
}
```

## 7. Backend implementation

### 7.1 Middleware order

```go
app := fiber.New(fiber.Config{ErrorHandler: ErrorHandler})
app.Use(RequestID())      // 1. first, always
app.Use(AccessLog())      // 2. one completion line per request
app.Use(recover.New(...)) // 3. panics become errors handled by ErrorHandler
// 4. CORS, security headers, body limit
// 5. session/auth, then tenant resolution (enrich logger with user_id, tenant_id)
// 6. routes
```

The recover middleware MUST log the stack (`debug.Stack()`) via its stack-trace handler (VERIFY `recover.Config` field names in v3) and return an error so the central handler produces the 500 body. Goroutines started outside the request (jobs, workers) are not covered by recover and MUST use a shared `safego` wrapper that recovers, logs at Error with stack and `request_id` if known.

### 7.2 `apperr` package

```go
package apperr

type Code string

type FieldError struct {
  Code    string `json:"code"`
  Message string `json:"message"`
}

type Error struct {
  Code       Code
  Message    string                  // client-safe; ignored for 5xx
  Fields     map[string][]FieldError // optional
  RetryAfter time.Duration           // optional, for 429/503
  Cause      error                   // logged, never sent to the client
}

func (e *Error) Error() string { /* Code + Message + Cause */ }
func (e *Error) Unwrap() error { return e.Cause }

func New(code Code, msg string) *Error
func Wrap(code Code, msg string, cause error) *Error
func Validation(fields map[string][]FieldError) *Error // VALIDATION_FAILED
func Register(code Code, status int)
func Status(code Code) int // unregistered code is a programming error caught by tests
```

Handlers and services return errors; they MUST NOT write error JSON themselves. Wrap lower-level errors with `%w` or `apperr.Wrap` so `errors.As` works.

### 7.3 Central error handler

Behavior requirements:

- Resolve with `errors.As` (never a bare type switch) in this order: `*apperr.Error`, `validator.ValidationErrors`, `*json.UnmarshalTypeError`, `*json.SyntaxError`, `*fiber.Error`, anything else (500).
- For status >= 500, force the generic message for the code and an empty `errors` map.
- Never name a local variable `errors` (shadows the package).
- Log once per error using the request-scoped logger and the level table in 8.3. For 4xx do not log the raw `err` (it may echo user input); log `code` and field keys only.
- Return a typed `ErrorResponse`.

```go
type ErrorResponse struct {
  Code      string                         `json:"code"`
  Message   string                         `json:"message"`
  Errors    map[string][]apperr.FieldError `json:"errors"`
  RequestID string                         `json:"request_id"`
}

func ErrorHandler(c fiber.Ctx, err error) error {
  ctx := c.Context()
  status := fiber.StatusInternalServerError
  resp := ErrorResponse{
    Code: "INTERNAL_ERROR", Errors: map[string][]apperr.FieldError{},
    RequestID: RequestIDFrom(ctx),
  }
  var (
    ae *apperr.Error
    ve validator.ValidationErrors
    te *json.UnmarshalTypeError
    se *json.SyntaxError
    fe *fiber.Error
  )
  switch {
  case errors.As(err, &ae):
    status, resp.Code, resp.Message = apperr.Status(ae.Code), string(ae.Code), ae.Message
    if ae.Fields != nil {
      resp.Errors = ae.Fields
    }
    if ae.RetryAfter > 0 {
      c.Set("Retry-After", strconv.Itoa(int(ae.RetryAfter.Seconds())))
    }
  case errors.As(err, &ve):
    status, resp.Code, resp.Message = 422, "VALIDATION_FAILED", "Request validation failed"
    resp.Errors = formatValidationErrors(ve)
  case errors.As(err, &te):
    status, resp.Code, resp.Message = 422, "VALIDATION_FAILED", "Request validation failed"
    resp.Errors = map[string][]apperr.FieldError{
      te.Field: {{Code: "invalid_type", Message: "has the wrong type"}},
    }
  case errors.As(err, &se):
    status, resp.Code, resp.Message = 400, "BAD_REQUEST", "Malformed JSON"
  case errors.As(err, &fe):
    status, resp.Code, resp.Message = fe.Code, statusToCode(fe.Code), fe.Message
  }
  if status >= 500 {
    resp.Message = defaultMessage(resp.Code)
    resp.Errors = map[string][]apperr.FieldError{}
  }

  lvl := slog.LevelInfo
  attrs := []any{"code", resp.Code, "status", status, "route", c.Route().Path}
  switch {
  case status >= 500:
    lvl = slog.LevelError
    attrs = append(attrs, "err", err)
  case status == 429:
    lvl = slog.LevelWarn
  }
  LoggerFrom(ctx).Log(ctx, lvl, "request error", attrs...)

  c.Set("Cache-Control", "no-store")
  return c.Status(status).JSON(resp)
}
```

### 7.4 Validation formatting

Register `TagNameFunc` so keys use JSON names (`name := strings.Split(f.Tag.Get("json"), ",")[0]`, falling back to the Go name when empty or `-`). `formatValidationErrors` strips the root struct name from `Namespace()`, maps each tag to a field code using the table in 5.4, and builds the default English message from code and tag param. Messages MUST NOT include the submitted value.

## 8. Logging

### 8.1 Setup

- `slog.NewJSONHandler(os.Stdout, ...)` in all deployed environments, text handler locally. The platform ships stdout; the app does not write log files.
- Call `slog.SetDefault` once at startup with static attributes `service`, `env`, `version` (git SHA).
- Log level is configurable by env var (default `info`).

### 8.2 Request-scoped logger

The RequestID middleware stores a logger carrying `request_id` in context. Code MUST obtain it with `LoggerFrom(ctx)` rather than adding `request_id` by hand. After auth and tenant resolution, middleware replaces it with `LoggerFrom(ctx).With("user_id", ..., "tenant_id", ...)`. Use the `...Context` slog methods.

### 8.3 Levels

| Situation | Level |
| --- | --- |
| 5xx, recovered panics, failed background jobs | Error |
| 429, repeated auth failures on sensitive routes | Warn |
| Other 4xx (validation, 401, 403, 404, 409) | Info |
| Request completion line | Info |

### 8.4 Access log

`AccessLog` emits exactly one line per request after the response status is final: `method`, `route` (pattern, not raw path), `status`, `duration_ms`, `bytes`, plus the context attributes. It MUST NOT emit a separate request-started line and SHOULD skip health endpoints. Because Fiber invokes the app `ErrorHandler` after the middleware chain unwinds, `AccessLog` should call the configured error handler itself when `c.Next()` returns an error, then return nil (VERIFY the v3 API for reading the handler, for example `c.App().Config().ErrorHandler`). Errored requests therefore produce one error line and one access line sharing `request_id`; this is accepted.

### 8.5 Standard attributes

`time, level, msg, service, env, version, request_id, tenant_id, user_id, method, route, path, status, duration_ms, code, err, stack`. Stack is present only on 5xx and panics.

### 8.6 Redaction

- MUST NOT log request or response bodies, `Cookie`, `Set-Cookie`, `Authorization`, session IDs, tokens, passwords.
- Types holding secrets implement `slog.LogValuer` and redact themselves.
- Use the route pattern for aggregation; raw path is secondary.
- Do not log wrapped validator errors for 4xx (they may include input values).

### 8.7 Debuggability without an error tracker

Plain `fmt.Errorf` errors carry no stack. For 5xx, the error chain (`%w`) is logged via `err`, recovered panics log `debug.Stack()`, and `apperr.Wrap` SHOULD record the operation name (for example `Wrap(code, msg, cause).Op("invite.create")`) so the failing layer is identifiable from the log line.

### 8.8 Alerting (required before launch)

At least one alert on 5xx rate and one on any Error-level line, using whatever aggregator is deployed. Logs nobody is alerted from are not a monitoring strategy.

## 9. Frontend (TanStack Start)

This section is framework-flavor agnostic (React or Solid). VERIFY all server-only APIs (`getRequestHeaders`, server middleware, server functions) against the installed Start version, and keep server-only code out of the client bundle.

### 9.1 Request ID on the Start server

For each incoming browser request, Start server code MUST read `X-Request-ID` or generate one in the same `req_` + 22 format, make it available to loaders and server functions, forward it on every call to the Go API, and set it on the page response. Browser-originated calls straight to the API omit the header; the Go API generates one. A single page load then traces through both tiers.

### 9.2 `ApiError`

```ts
export interface FieldError { code: string; message: string }

export class ApiError extends Error {
  readonly status: number            // 0 for network failures
  readonly code: string
  readonly fieldErrors: Record<string, FieldError[]>
  readonly requestId?: string
  readonly retryAfterSeconds?: number

  constructor(init: {
    status: number; code: string; message: string
    fieldErrors?: Record<string, FieldError[]>
    requestId?: string; retryAfterSeconds?: number; cause?: unknown
  }) {
    super(init.message, { cause: init.cause })
    this.name = 'ApiError'
    this.status = init.status
    this.code = init.code
    this.fieldErrors = init.fieldErrors ?? {}
    this.requestId = init.requestId
    this.retryAfterSeconds = init.retryAfterSeconds
  }
}

// Class identity may not survive router/SSR serialization, so also match by shape.
export function isApiError(e: unknown): e is ApiError {
  return e instanceof ApiError ||
    (typeof e === 'object' && e !== null &&
      (e as { name?: unknown }).name === 'ApiError' &&
      typeof (e as { status?: unknown }).status === 'number')
}
```

### 9.3 `apiFetch`

Requirements:

- Base URL: relative `/api/v1` in the browser; absolute `API_INTERNAL_URL` on the server (relative URLs fail in loaders).
- On the server, forward the incoming `Cookie` header and `X-Request-ID` (the `credentials` option does nothing server-side).
- Merge headers with `new Headers(init.headers)` so caller headers never clobber defaults.
- Normalize every failure into `ApiError`: JSON error body, non-JSON body from a proxy or load balancer (derive `code` from status), network `TypeError` (`status: 0`, `code: 'NETWORK_ERROR'`), and timeouts. Rethrow aborts untouched.
- Always read `X-Request-ID` from the response header as a fallback for `requestId`; parse `Retry-After`.
- Handle 204.

```ts
export async function apiFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
  const isServer = typeof window === 'undefined'
  const headers = new Headers(init.headers)
  if (init.body !== undefined && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }
  if (isServer) {
    for (const [k, v] of await serverForwardHeaders()) headers.set(k, v) // Cookie, X-Request-ID
  }
  let res: Response
  try {
    res = await fetch(`${apiBase(isServer)}${path}`, { ...init, headers, credentials: 'include' })
  } catch (cause) {
    if (init.signal?.aborted) throw cause
    throw new ApiError({ status: 0, code: 'NETWORK_ERROR', message: 'Network error', cause })
  }
  if (!res.ok) {
    let body: Partial<{ code: string; message: string; errors: Record<string, FieldError[]>; request_id: string }> = {}
    try { body = await res.json() } catch { /* non-JSON error body */ }
    throw new ApiError({
      status: res.status,
      code: body.code ?? codeFromStatus(res.status),
      message: body.message ?? res.statusText,
      fieldErrors: body.errors ?? {},
      requestId: body.request_id ?? res.headers.get('X-Request-ID') ?? undefined,
      retryAfterSeconds: parseRetryAfter(res.headers.get('Retry-After')),
    })
  }
  if (res.status === 204) return undefined as T
  return (await res.json()) as T
}
```

If the API is ever served from a different origin, it MUST send `Access-Control-Expose-Headers: X-Request-ID, Retry-After`, otherwise `res.headers.get` returns null.

### 9.4 Consumption rules

- Branch on `code`, never `message`. Localize by `code` (and field `code`) where translations exist.
- Forms: map `fieldErrors` onto inputs by key (TanStack Form server errors or equivalent); show `_form` errors and the top-level `message` in a form-level banner for 422 and 409.
- Show the server `message` only for 4xx. For 5xx and network errors show a generic message plus the reference `request_id` with a copy button.
- The old `getErrorMessage` is replaced by helpers that return `{ banner, fields }`. It MUST NOT print raw field names or discard `message`.

### 9.5 Global policies

- 401: one central handler (Query cache `onError` or Router) redirects to login preserving the return URL; excluded for the login and session-check calls.
- `CSRF_FAILED`: refresh the token and retry once.
- TanStack Query: no retries for 4xx except 408 and 429 (honor `retryAfterSeconds`); at most 2 retries for 5xx and network failures on queries; mutations never auto-retry.
- Do not report 4xx to any console or telemetry sink. In development, `console.error` 5xx with `requestId`.

### 9.6 Error boundary

The root and route error boundaries render a friendly message and, for any `ApiError` with a `requestId` (or any error with a request ID), a visible reference the user can copy for support. They MUST handle errors that arrive serialized from SSR (use `isApiError`).

## 10. Security and privacy

- Cross-tenant and non-existent resources both return 404 with identical bodies.
- Auth failures use generic messages (no account enumeration).
- Error bodies and logs contain no PII beyond IDs; no secrets ever.
- Stack traces and internal errors never leave the server.
- Inbound `X-Request-ID` is validated (section 6).

## 11. Testing requirements

Backend contract tests (table-driven, through the real app, for example `app.Test`):

| Case | Expected |
| --- | --- |
| Malformed JSON | 400 `BAD_REQUEST` |
| Missing required field | 422, key = JSON name, field code `required` |
| JSON type mismatch | 422, field code `invalid_type` |
| Nested and array field | key like `items[0].name` |
| No session / forbidden / cross-tenant | 401 / 403 / 404 |
| Unknown route, wrong method | 404 / 405 with the full JSON shape and a request ID |
| Body over limit | 413 `PAYLOAD_TOO_LARGE` |
| Handler panic | 500, generic message, stack only in logs |
| `apperr` with `Cause` | 500 body excludes cause text |
| Valid inbound `X-Request-ID` | echoed in header and body |
| Invalid inbound ID | replaced; header equals body `request_id` |
| 429 | `Retry-After` set |
| Any error | `errors` is an object, never null |
| Registry | every code has a status; every status is 4xx or 5xx |
| Logs | error line has `request_id`, correct level, no `Cookie` or `Authorization` |

Frontend tests: `apiFetch` normalization (JSON error, HTML 502, network failure, abort, 204); server-side call forwards `Cookie` and `X-Request-ID`; `isApiError` after serialization round trip; field mapping to a form; boundary shows request ID; query retry policy; 401 redirect.

CI guardrails: fail the build on `fiber.Map` error responses or `c.Status(4xx|5xx).JSON(` outside the httpserver package.

## 12. Migration and rollout

- Frontend and backend deploy together, so a single cutover is acceptable; avoid a long dual-shape period.
- Land the central handler first. Handlers that still write their own error JSON are migrated module by module (one spec per module).
- First task of the handler migration: inventory every existing error code (for example `INVALID_EMAIL`). Each becomes a registered domain code if the frontend relies on it, or maps to `VALIDATION_FAILED` plus a field error.
- During migration the frontend normalizer tolerates the legacy shape (missing `errors` or `request_id`; string field errors converted to `{code: 'invalid', message}`). Remove this tolerance in the final phase.
- Ship an OpenAPI schema for `ErrorResponse` and generate the TypeScript types from it so the contract cannot drift.

## 13. Implementation plan

Each row becomes one downstream spec. Acceptance criteria reference sections of this document.

| ID | Phase | Deliverable | Acceptance | Depends on | Status |
| --- | --- | --- | --- | --- | --- |
| P1 | apperr and registry | `apperr` package, code registry, `ErrorResponse` types | 5.1, 5.2, 7.2; registry tests pass | none | Not started |
| P2 | Request ID and logger | `RequestID` middleware, context helpers, middleware order | 6, 7.1, 8.2 | P1 | Not started |
| P3 | Central handler | `ErrorHandler`, recover integration, validation formatting | 4, 5.3, 5.4, 7.3, 7.4 | P1, P2 | Not started |
| P4 | Contract tests | Full backend table from section 11, written alongside P3 | All rows green | P3 | Not started |
| P5 | Logging | slog setup, `AccessLog`, level policy, redaction, `safego`, alert definitions | 8.1 to 8.8 | P2 | Not started |
| P6 | Handler migration | Per-module specs converting handlers to `apperr`; legacy code inventory | 7.2, 12; no ad hoc error JSON | P3, P4 | Not started |
| P7 | Frontend client | `ApiError`, `apiFetch`, SSR cookie and request ID forwarding, Start request ID | 9.1 to 9.3 | P3 | Not started |
| P8 | Frontend UX | Form mapping, global policies, error boundary, replace `getErrorMessage` | 9.4 to 9.6 | P7 | Not started |
| P9 | Hardening | CI guardrails, OpenAPI-generated types, remove legacy tolerance | 11, 12 | P6, P8 | Not started |

## 14. Future work and open questions

- **Error tracker (Sentry or similar):** deferred. The reserved hook point is the `status >= 500` branch of `ErrorHandler` plus the recover path; tag events with `request_id`. Nothing else in this design changes. The official Sentry Go integration for Fiber v3 is the `sentry-go/fiberv3` package, and its middleware must run after `RequestID`.
- **OpenTelemetry:** if adopted, align `request_id` with the W3C `traceparent` trace ID.
- **Open questions for the owner:** (1) React or Solid for TanStack Start; (2) which log aggregator is deployed; (3) does Start proxy `/api` or does the browser call the Go API directly (affects D7 and CORS); (4) whether the frontend needs i18n for error messages at launch.

## Related

- Error Contract Skill
- Backend AGENTS.md BE-025
- ADR-0002 Auth and Roles