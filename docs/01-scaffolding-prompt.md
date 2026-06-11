# Somotracker Go Backend — Scaffolding Prompt (Enhanced)

You are a Staff Backend Engineer specialized in Go (Fiber), multi-process single-binary architecture, and high-performance containerized setups.

We are initializing the Go backend engine for **Somotracker** under the `./backend/` path in a monorepo workspace. The architecture must strictly abide by the rules established in `./backend/AGENTS.md`: high-ROI, waste-free patterns, absolute Locality of Behavior (LoB), non-negotiable multi-process graceful shutdowns, and rigorous compilation safety.

We are building the foundational runtime stack including Docker Compose local orchestration with hot-reloading (via Air), Postgres connection pooling (via pgx/v5), Redis, an initialized empty tenant domain module, and a global system health check.

**CRITICAL:** Use Uber's `fx` package for structured, type-safe Dependency Injection from the get-go. No manual constructor wiring is allowed inside `main.go`. Everything must flow through modular `fx.Provide` and `fx.Invoke` structures.

**COMPILATION GATE:** Every file generated must compile cleanly via `go build ./...` with zero errors and zero unused imports. All `fx`-provided types must be consumed somewhere in the container — unused provisions must be wired through `fx.Invoke` or removed. No `_` import aliases unless explicitly required by a side-effect-only package.

---

## 1. Dependencies setup

Add and install the following packages:

```
github.com/gofiber/fiber/v2
github.com/jackc/pgx/v5/pgxpool
github.com/redis/go-redis/v9
go.uber.org/fx
```

---

## 2. Docker orchestration: `./docker-compose.yml`

Create `docker-compose.yml` in the root workspace folder with three services:

**`somotracker_postgres`** — `postgres:16-alpine`
- Database: `somotracker_dev`, user: `somo_admin`, password: `somo_secure_password`
- Healthcheck using `pg_isready -U somo_admin -d somotracker_dev`
- Named volume for data persistence: `postgres_data:/var/lib/postgresql/data`

**`somotracker_redis`** — `redis:7-alpine`
- Healthcheck using `redis-cli ping`
- Named volume for AOF persistence: `redis_data:/data`

**`somotracker_api`** — `cosmtrek/air:v1.51.0`
- `working_dir: /app`
- Volume mount: `./backend:/app`
- Expose port `3030`
- `depends_on` both database services with condition `service_healthy`
- Pass `DATABASE_URL` and `REDIS_URL` as environment variables that match their service hostnames (i.e. `postgres:5432` and `redis:6379`)

Declare all named volumes at the bottom of the file under a top-level `volumes:` key.

---

## 3. Global application config: `./backend/internal/config/config.go`

Create a clean environment parser using `os.Getenv` with safe fallbacks:

| Env variable | Fallback |
|---|---|
| `DATABASE_URL` | `postgres://somo_admin:somo_secure_password@postgres:5432/somotracker_dev?sslmode=disable` |
| `REDIS_URL` | `redis:6379` |
| `APP_ENV` | `development` |
| `PORT` | `3030` |

Expose a `Config` struct holding these four fields. Expose a `Load() Config` constructor function. Wrap it in an `fx`-friendly provider:

```go
var Module = fx.Provide(Load)
```

---

## 4. Storage pools: `./backend/internal/database/database.go`

Create a thread-safe storage connection factory exposing a unified `Pools` struct with two fields: `PG *pgxpool.Pool` and `Redis *redis.Client`.

**PostgreSQL** — use `pgxpool.ParseConfig`, then set on the resulting config:
- `MaxConns = 25`
- `MinConns = 5`
- `MaxConnLifetime = 30 * time.Minute`
- `MaxConnIdleTime = 5 * time.Minute`

After creating the pool, call `pool.Ping(ctx)` to validate connectivity at startup. If the ping fails, return a wrapped error with context (e.g. `fmt.Errorf("postgres ping: %w", err)`).

**Redis** — configure using `redis.Options{Addr: cfg.RedisURL}`. After creating the client, call `client.Ping(ctx)` to validate connectivity. If the ping fails, return a wrapped error.

Expose a `Connect(cfg config.Config) (*Pools, error)` constructor and wrap it in:

```go
var Module = fx.Provide(Connect)
```

---

## 5. Security middleware pipeline: `./backend/internal/middleware/security.go`

Create a global security interceptor implementing five layers. Expose a single setup function:

```go
func Register(app *fiber.App, pools *database.Pools)
```

Mount layers in this order:

### Layer 1 — Panic recovery
Use `fibermiddleware.New()` (Fiber's built-in recover middleware). This must be the first middleware registered so it catches panics from all subsequent layers.

### Layer 2 — Security headers
On every response, inject these headers via a `app.Use` handler:
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Content-Security-Policy: default-src 'self'`

### Layer 3 — Custom header CSRF guard
> **Important design note:** This is a defense-in-depth layer, not a replacement for token-based CSRF protection. It guards against simple cross-origin form submissions but is bypassable by CORS-misconfigured endpoints. Because this project uses server-side sessions (not JWTs), the session cookie's `SameSite=Strict` flag provides strong CSRF protection for same-site navigation. This header guard adds defense-in-depth for cross-origin API consumers.

For mutating methods (`POST`, `PUT`, `DELETE`, `PATCH`): if the `X-Requested-With` header is absent, halt and return:
```json
{ "error": "forbidden", "reason": "missing X-Requested-With header" }
```
with HTTP status `403`.

### Layer 4 — Redis sliding-window rate limiter
Implement a true sliding-window rate limiter using a Redis sorted set per IP. The algorithm must be atomic — use a single Lua script executed via `redis.Client.Eval`:

```lua
local key    = KEYS[1]
local now    = tonumber(ARGV[1])   -- current unix milliseconds
local window = tonumber(ARGV[2])   -- window size in milliseconds (60000)
local limit  = tonumber(ARGV[3])   -- max requests (60)
local id     = ARGV[4]             -- unique request ID (UUID or nanosecond timestamp string)

redis.call('ZREMRANGEBYSCORE', key, '-inf', now - window)
local count = redis.call('ZCARD', key)
if count >= limit then
  return 0
end
redis.call('ZADD', key, now, id)
redis.call('PEXPIRE', key, window)
return 1
```

Key format: `ratelimit:{ip}`. Window: 60,000 ms. Limit: 60 requests. On rejection, return HTTP `429` with header `Retry-After: 60` and body:
```json
{ "error": "rate_limit_exceeded", "retry_after_seconds": 60 }
```

### Layer 5 — Device fingerprinting
Read `c.IP()`, `c.Get("User-Agent")`, and `c.Get("Accept-Language")`. Concatenate them with a `|` separator. Hash using `crypto/sha256`.

> **Implementation note:** SHA-256 is used here for its collision resistance properties, not raw speed. If sub-microsecond fingerprinting is required at scale, consider replacing with `fnv.New64a()` from `hash/fnv` (non-cryptographic, ~5× faster). The current choice is deliberately conservative for a security-adjacent context.

Store the hex-encoded result in Fiber locals:
```go
c.Locals("device_fingerprint", fingerprintHex)
```

---

## 6. SSRF-safe HTTP utility client: `./backend/internal/utils/http_client.go`

Create an internal utility package providing a safe outbound HTTP client that blocks requests to private infrastructure.

Use a custom `net.Dialer` with a `Control` function. Inside the control function:

1. Parse the target address to extract the IP.
2. Resolve the hostname to its IP address(es) **after** DNS resolution (i.e. operate on the resolved IP, not the hostname string) to prevent DNS rebinding attacks where an attacker's DNS resolves to a private IP on second lookup.
3. Block the connection if the resolved IP matches any of:
   - Loopback: `127.0.0.0/8` and `::1/128`
   - Private class A: `10.0.0.0/8`
   - Private class B: `172.16.0.0/12`
   - Private class C: `192.168.0.0/16`
   - Link-local / metadata: `169.254.0.0/16` (covers `169.254.169.254` cloud metadata endpoints)
   - IPv6 link-local: `fe80::/10`

Return a typed sentinel error on block:
```go
var ErrSSRFBlocked = errors.New("ssrf: connection to private/loopback address blocked")
```

Wrap the dialer inside a `&http.Transport{}` and expose a `*http.Client` with a 10-second timeout.

Provide via `fx`:
```go
var Module = fx.Provide(NewSafeClient)
```

**Wire it into the container** via `fx.Invoke` in `main.go` so the `fx` container does not flag it as an unused provision (even if no domain module consumes it yet).

---

## 7. Tenant domain module: `./backend/internal/tenant/`

Set up a clean functional layout using four files. Each file has one responsibility.

### `domain.go`
```go
package tenant

type Tenant struct {
    ID        string    `db:"id"         json:"id"`
    Name      string    `db:"name"       json:"name"`
    Slug      string    `db:"slug"       json:"slug"`
    CreatedAt time.Time `db:"created_at" json:"created_at"`
}
```

### `repository.go`
Declare the port interface and a concrete adapter:
```go
type Repository interface {
    // placeholder — to be expanded when schema is finalized
}

type SqlcRepository struct {
    pools *database.Pools
}

func NewRepository(pools *database.Pools) *SqlcRepository {
    return &SqlcRepository{pools: pools}
}
```

### `service.go`
```go
type Service struct {
    repo Repository
}

func NewService(repo *SqlcRepository) *Service {
    return &Service{repo: repo}
}
```

The `Service` constructor accepts the concrete `*SqlcRepository` for now, not the interface, to avoid an `fx` ambiguous-binding error. This trades a small testability cost for a working DI graph at this scaffolding stage. Refactor to interface injection once the first real method is added.

### `handler.go`
```go
type Handler struct {
    svc *Service
}

func NewHandler(svc *Service) *Handler {
    return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(router fiber.Router) {
    tenants := router.Group("/tenants")
    _ = tenants // placeholder — routes will be registered here
}
```

At the bottom of `handler.go`, declare the module:
```go
var Module = fx.Module("tenant",
    fx.Provide(
        NewRepository,
        NewService,
        NewHandler,
    ),
)
```

---

## 8. Application entrypoint: `./backend/cmd/api/main.go`

Write the full lifecycle bootstrap using `fx.New`. No manual constructor calls. No `init()` functions.

```go
func main() {
    fx.New(
        config.Module,
        database.Module,
        utils.Module,   // SSRF-safe HTTP client
        tenant.Module,

        fx.Invoke(registerApp),
        fx.Invoke(consumeSafeClient), // prevents unused-provision warning
    ).Run()
}
```

### `registerApp` function signature
```go
func registerApp(
    lc fx.Lifecycle,
    cfg config.Config,
    pools *database.Pools,
    tenantHandler *tenant.Handler,
)
```

### `OnStart` hook — execute in this exact order:
1. Create `fiber.App` with a `fiber.Config{AppName: "somotracker"}`.
2. Call `middleware.Register(app, pools)` to mount the full security pipeline.
3. Register the global health endpoint:
   ```go
   app.Get("/health", func(c *fiber.Ctx) error {
       ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
       defer cancel()
       pgErr  := pools.PG.Ping(ctx)
       redErr := pools.Redis.Ping(ctx).Err()
       return c.JSON(fiber.Map{
           "status":    "ok",
           "postgres":  errToStatus(pgErr),
           "redis":     errToStatus(redErr),
           "env":       cfg.AppEnv,
       })
   })
   ```
4. Call `tenantHandler.RegisterRoutes(app)` to mount domain routes.
5. Start Fiber in a non-blocking goroutine:
   ```go
   go func() {
       if err := app.Listen(":" + cfg.Port); err != nil && !errors.Is(err, http.ErrServerClosed) {
           log.Fatalf("fiber listen: %v", err)
       }
   }()
   ```

### `OnStop` hook — execute in this exact order:
1. Create a `context.WithTimeout` of **15 seconds** to bound the total shutdown window.
2. Call `app.ShutdownWithContext(ctx)` — this is Fiber's graceful shutdown that waits for in-flight requests to complete within the context deadline. Do **not** call the plain `app.Shutdown()`, which has no deadline.
3. After Fiber drains, close the Postgres pool: `pools.PG.Close()`.
4. After Postgres closes, close the Redis client: `pools.Redis.Close()`.
5. If any step returns an error, log it but do not suppress subsequent cleanup steps (use a multi-error approach).

### Helper — `errToStatus`
```go
func errToStatus(err error) string {
    if err == nil {
        return "healthy"
    }
    return "unhealthy: " + err.Error()
}
```

### Helper — `consumeSafeClient`
```go
func consumeSafeClient(client *http.Client) {
    // intentional no-op: ensures the SSRF-safe client is wired into
    // the fx container so it is available to future consumers without
    // triggering an unused-provision warning.
    _ = client
}
```

---

---

## 9. Next.js frontend security: `./frontend/`

> **Scope note:** This section covers the security boundary between a Next.js 14+ (App Router) frontend and the Go Fiber backend. Every item here is a direct consequence of having two separate origins in the same product. Treat each sub-section as a standalone prompt you can hand to a frontend-focused agent.

---

### 9.1 CORS policy on the Go backend: `./backend/internal/middleware/cors.go`

Adding a browser-facing frontend means the backend must now express an explicit CORS policy. Add a `cors.go` file to the middleware package and register it inside `middleware.Register()` **before** all other layers (even panic recovery) so preflight `OPTIONS` requests are handled before any security logic runs.

Install the Fiber CORS middleware:
```
github.com/gofiber/fiber/v2/middleware/cors
```

Configure it as follows:

```go
app.Use(cors.New(cors.Config{
    AllowOrigins:     cfg.AllowedOrigins, // e.g. "http://localhost:3000" in dev, real domain in prod
    AllowMethods:     "GET,POST,PUT,DELETE,PATCH,OPTIONS",
    AllowHeaders:     "Origin,Content-Type,Accept,Authorization,X-Requested-With",
    AllowCredentials: true,
    MaxAge:           86400, // cache preflight for 24 hours
}))
```

Add `ALLOWED_ORIGINS` to `config.Config` with fallback `http://localhost:3000`.

**Critical rules:**
- Never set `AllowOrigins: "*"` when `AllowCredentials: true` — browsers will block the response and this combination is invalid per the CORS spec.
- In production, `AllowedOrigins` must be set to the exact frontend domain(s) from an environment variable. Wildcard origins are forbidden in production.
- The `OPTIONS` method must be included in `AllowMethods` or preflight requests will return `405` and silently break all non-simple requests from the browser.

Add to the verification checklist: a browser `fetch` with `credentials: "include"` from `localhost:3000` to `localhost:3030/health` must return `200` without a CORS error.

---

### 9.2 Redis-backed session + HttpOnly cookie auth strategy: `./frontend/lib/auth.ts` and `./backend/internal/middleware/auth.go`

Next.js introduces two common session storage mistakes. Specify the correct strategy explicitly so neither agent makes the wrong choice.

**What to build on the backend (`auth.go`):**

Create a `ValidateSession` middleware that:
1. Reads the session ID exclusively from the `HttpOnly` cookie named `somo_session` — never from `Authorization: Bearer` headers for browser-initiated requests (Bearer tokens are for M2M/API clients only, not browsers).
2. Looks up the session ID in Redis using key format `session:{id}`. The stored value is a JSON blob containing `user_id`, `tenant_id`, and `role`.
3. On a cache hit, deserialise the session data and store it in Fiber locals (`c.Locals("session", sessionData)`). Also call `Redis.Expire` to implement a rolling session window (e.g. 24 hours from last activity).
4. On a cache miss or any Redis error, return `401` with body `{"error": "unauthorized"}`. Never reveal whether the session was missing vs expired vs invalid — all map to `401`.
5. Session IDs must be cryptographically random — generate with `crypto/rand`, 32 bytes, hex-encoded (64 chars). Never use sequential or predictable IDs.

**What to build on the frontend (`lib/auth.ts`):**

When the backend creates a session at login:
- The backend stores session data in Redis and issues the session ID via `Set-Cookie` with flags: `HttpOnly; Secure; SameSite=Strict; Path=/; Max-Age=86400`.
- Never write the session ID to `localStorage` or `sessionStorage` — these are readable by any JavaScript on the page, including injected XSS payloads.
- Never write the session ID to a non-`HttpOnly` cookie — React/Next.js code should never be able to call `document.cookie` and read it.
- On logout, the backend must actively delete the Redis session key (`DEL session:{id}`) before expiring the cookie. Cookie deletion alone is not sufficient — the server-side session must be invalidated.

**Next.js middleware (`./frontend/middleware.ts`):**

Write a Next.js edge middleware that checks for the presence of the `somo_session` cookie on every protected route. If absent, redirect to `/login`. This is a UX guard only — it is not a security boundary. The Go backend `ValidateSession` middleware is the actual security gate.

```typescript
import { NextRequest, NextResponse } from 'next/server'

const PROTECTED_PREFIXES = ['/dashboard', '/settings', '/admin']

export function middleware(req: NextRequest) {
  const isProtected = PROTECTED_PREFIXES.some(p => req.nextUrl.pathname.startsWith(p))
  if (isProtected && !req.cookies.get('somo_session')) {
    return NextResponse.redirect(new URL('/login', req.url))
  }
  return NextResponse.next()
}

export const config = { matcher: ['/((?!_next/static|_next/image|favicon.ico).*)'] }
```

---

### 9.3 Environment variable discipline: `./frontend/.env.local` and `./frontend/next.config.ts`

Next.js has two categories of environment variables with critically different exposure:

| Prefix | Exposed to | Risk if misused |
|---|---|---|
| `NEXT_PUBLIC_` | Browser JS bundle | Any user can read it in DevTools |
| (no prefix) | Server-side only (SSR, API routes, RSC) | Never reaches the browser |

**Rules to enforce — add as a lint comment block at the top of `next.config.ts`:**

```typescript
// SECURITY RULES — DO NOT VIOLATE:
// 1. NEXT_PUBLIC_ variables must contain ZERO secrets.
//    Acceptable: API base URL, feature flags, analytics IDs.
//    Never: session secrets, DB URLs, API keys, service account credentials.
// 2. The backend DATABASE_URL and REDIS_URL must never appear in this file.
// 3. Internal session secrets or signing keys must never appear in this file under any name.
// 4. Server-only secrets go in .env.local (gitignored) and are accessed
//    only inside Server Components, Route Handlers, or getServerSideProps.
// 5. Run `npx @next/codemod` or manually audit: grep -r "NEXT_PUBLIC_" ./
//    and verify each result contains no credentials.
```

Add to `.gitignore`:
```
.env.local
.env.*.local
```

Add a CI check that fails if any `NEXT_PUBLIC_` variable value matches a secret pattern (JWT, private key, connection string). A simple shell check:
```bash
grep -r "NEXT_PUBLIC_" .env* | grep -E "(password|secret|key|session|DATABASE_URL|REDIS)" && echo "SECRET LEAK" && exit 1
```

---

### 9.4 Content Security Policy for Next.js RSC and chunks: `./frontend/next.config.ts`

The `Content-Security-Policy` header set by the Go backend (`default-src 'self'`) will break Next.js in production because:
- Next.js lazy-loads JS chunks from `/_next/static/chunks/` — these are same-origin but the dynamic import pattern may require `script-src 'self' 'unsafe-eval'` in development (Turbopack).
- React Server Components stream inline `<script>` tags with nonces for hydration.
- Third-party fonts, analytics, or image domains need explicit allowlisting.

**Two required changes:**

**On the Go backend**, scope the `Content-Security-Policy` header to API responses only (i.e. routes under `/api/` and `/health`), not to all responses. The frontend's Next.js config will own the CSP for HTML page responses:

```go
// In security headers middleware — scope to API paths only
if strings.HasPrefix(c.Path(), "/api/") || c.Path() == "/health" {
    c.Set("Content-Security-Policy", "default-src 'self'")
}
```

**On the Next.js frontend**, implement a nonce-based CSP in `next.config.ts` using the `headers()` config:

```typescript
// next.config.ts
import type { NextConfig } from 'next'
import crypto from 'crypto'

const nextConfig: NextConfig = {
  async headers() {
    const nonce = crypto.randomBytes(16).toString('base64')
    return [
      {
        source: '/(.*)',
        headers: [
          {
            key: 'Content-Security-Policy',
            value: [
              `default-src 'self'`,
              `script-src 'self' 'nonce-${nonce}' 'strict-dynamic'`,
              `style-src 'self' 'unsafe-inline'`,   // Next.js requires this for CSS-in-JS
              `img-src 'self' data: blob:`,
              `font-src 'self'`,
              `connect-src 'self' ${process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:3030'}`,
              `frame-ancestors 'none'`,              // redundant with X-Frame-Options but belt+suspenders
              `base-uri 'self'`,
              `form-action 'self'`,
            ].join('; '),
          },
          { key: 'X-Frame-Options', value: 'DENY' },
          { key: 'X-Content-Type-Options', value: 'nosniff' },
          { key: 'Referrer-Policy', value: 'strict-origin-when-cross-origin' },
          { key: 'Permissions-Policy', value: 'camera=(), microphone=(), geolocation=()' },
        ],
      },
    ]
  },
}

export default nextConfig
```

> **Implementation note:** The `nonce` approach above generates one nonce per build, not per request. For a per-request nonce (stronger), implement this in `./frontend/middleware.ts` using `NextResponse.next({ headers: { 'x-nonce': nonce } })` and read it inside a Server Component via `headers()`. This is the Next.js 14 recommended pattern — use it if the product handles sensitive data.

---

### 9.5 Open redirect guard: `./frontend/lib/redirect.ts`

Next.js `redirect()` and `router.push()` are commonly abused in phishing attacks when redirect targets are constructed from user-supplied query parameters (e.g. `/login?next=/dashboard` → manipulated to `/login?next=https://evil.com`).

Create a whitelist-based redirect sanitiser used everywhere a redirect target comes from user input:

```typescript
// ./frontend/lib/redirect.ts

const ALLOWED_REDIRECT_PREFIXES = [
  '/dashboard',
  '/settings',
  '/tenants',
]

/**
 * Sanitises a redirect target from user-supplied input.
 * Only allows relative paths on the allowlist. Returns /dashboard as the
 * safe default for anything that doesn't match.
 */
export function sanitiseRedirect(raw: string | null | undefined): string {
  if (!raw) return '/dashboard'

  // Reject anything that looks absolute (has a protocol or starts with //)
  if (/^https?:\/\//i.test(raw) || raw.startsWith('//')) return '/dashboard'

  // Reject anything with a newline (response splitting defence)
  if (raw.includes('\n') || raw.includes('\r')) return '/dashboard'

  const normalised = decodeURIComponent(raw).split('?')[0]
  const isAllowed = ALLOWED_REDIRECT_PREFIXES.some(p => normalised.startsWith(p))
  return isAllowed ? raw : '/dashboard'
}
```

Use `sanitiseRedirect(searchParams.get('next'))` everywhere a post-login or post-action redirect is performed. Never pass `router.push(searchParams.get('next'))` directly.

---

### 9.6 Supply chain: `./frontend/package.json` and CI

Next.js projects accumulate hundreds of transitive npm dependencies. Add the following hardening measures:

**Lock file enforcement** — add to CI:
```bash
npm ci --ignore-scripts   # use ci not install; --ignore-scripts prevents postinstall attacks
```

**Dependency audit gate** — add to CI as a required step:
```bash
npm audit --audit-level=high   # fails CI on high/critical CVEs
```

**Subresource Integrity for external scripts** — if any third-party `<script>` tags are added to `./frontend/app/layout.tsx`, they must include `integrity` and `crossOrigin="anonymous"` attributes:
```tsx
<script
  src="https://example.com/analytics.js"
  integrity="sha384-<hash>"
  crossOrigin="anonymous"
/>
```

**`package.json` engine pinning** — add to prevent running with unexpected Node versions:
```json
"engines": {
  "node": ">=20.0.0 <22.0.0",
  "npm": ">=10.0.0"
}
```

**`.npmrc`** — add to the frontend root to prevent dependency confusion attacks where an attacker publishes a malicious package to the public registry with the same name as an internal package:
```
registry=https://registry.npmjs.org/
save-exact=true
```

---

### 9.7 Docker Compose additions for the frontend service

Add a `somotracker_frontend` service to `./docker-compose.yml`:

```yaml
somotracker_frontend:
  image: node:20-alpine
  working_dir: /app
  volumes:
    - ./frontend:/app
    - /app/node_modules       # isolate container node_modules from host
    - /app/.next              # isolate build cache from host
  ports:
    - "3000:3000"
  environment:
    - NODE_ENV=development
    - NEXT_PUBLIC_API_URL=http://localhost:3030
  command: sh -c "npm ci --ignore-scripts && npm run dev"
  depends_on:
    - somotracker_api
```

The `node_modules` and `.next` anonymous volumes prevent the container from using the host's `node_modules` (which may have been installed with a different OS/architecture) and keep Next.js build artefacts from leaking to the host filesystem.

---

### Updated verification checklist additions (Next.js)

In addition to the Go backend checks, verify:

- `OPTIONS http://localhost:3030/api/tenants` from browser origin `http://localhost:3000` returns `200` with correct `Access-Control-Allow-Origin` and `Access-Control-Allow-Credentials: true`
- `POST http://localhost:3030/api/tenants` with `credentials: "include"` and a valid `somo_session` cookie succeeds; without the cookie, returns `401`
- `document.cookie` in the browser DevTools console does **not** show `somo_session` (confirming `HttpOnly` is set)
- `NEXT_PUBLIC_` grep across `./frontend/.env*` finds no secrets
- `npm audit --audit-level=high` exits `0` in CI
- `/login?next=https://evil.com` redirects to `/dashboard`, not to `evil.com`
- Browser DevTools Network tab shows `Content-Security-Policy` header on HTML responses from Next.js
- Chrome DevTools Console shows zero CSP violation warnings on page load

---

## Constraints and verification checklist

Before considering the scaffold complete, verify all of the following:

- `go build ./...` passes with zero errors
- `go vet ./...` passes with zero warnings
- No circular imports between packages (`config`, `database`, `middleware`, `utils`, `tenant`, `cmd/api`)
- Every `fx.Provide`d type is consumed by at least one `fx.Invoke` or downstream provider
- The `docker-compose up` sequence starts `somotracker_api` only after both database healthchecks pass
- `GET /health` returns `200` with `postgres: healthy` and `redis: healthy` when both services are reachable
- `POST /any-route` without `X-Requested-With` returns `403`
- Repeated rapid `GET` requests from a single IP eventually return `429`
- Graceful shutdown (`SIGTERM`) allows in-flight requests to complete before pool teardown


---

## Package dependency graph (no cycles allowed)

```
cmd/api
  ├── config
  ├── database       (depends on: config)
  ├── middleware      (depends on: database, config)
  ├── utils
  └── tenant
        ├── database
        └── (fiber.Router — injected at route registration, not package import)

frontend/                          (independent process, separate dependency tree)
  ├── lib/auth.ts                  (session cookie reads only — no direct backend import)
  ├── lib/redirect.ts              (pure utility — zero external deps)
  ├── middleware.ts                (Next.js edge runtime — no Node APIs)
  └── next.config.ts               (build-time only)
```

`config` and `utils` must have zero internal dependencies on other workspace packages. The `frontend/` tree must never import from `backend/` — the only coupling is the HTTP contract (JSON over REST) and the shared cookie name `somo_session`.