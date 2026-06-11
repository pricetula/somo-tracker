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
- Expose port `8080`
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
| `PORT` | `8080` |

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
> **Important design note:** This is a defense-in-depth layer, not a replacement for token-based CSRF protection. It guards against simple cross-origin form submissions but is bypassable by CORS-misconfigured endpoints. A full CSRF implementation (double-submit cookie or signed token) should be layered on top in a future iteration.

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
  ├── database   (depends on: config)
  ├── middleware  (depends on: database)
  ├── utils
  └── tenant
        ├── database
        └── (fiber.Router — injected at route registration, not package import)
```

`config` and `utils` must have zero internal dependencies on other workspace packages.