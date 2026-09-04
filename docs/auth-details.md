# Sign-Up Flow & Authentication Strategy

## 1. The Sign-Up Flowchart & Lifecycle

```
[Frontend] --(Email)--> [Backend API] --(Send Magic Link)--> [Stytch B2B API]

[Frontend] <--(Redirect / Dashboard)-- [Backend Cookie Set] <--(Callback + Intermediate Token)-- [Stytch Email Link Click]
```

- **Step 1: Initiation** — The user submits their work email on the frontend. The backend receives the request and calls the Stytch B2B Magic Link send endpoint.
- **Step 2: Dispatch** — Stytch emails the magic link to the user containing a secure, time-sensitive token.
- **Step 3: Verification & Redirect** — The user clicks the link. Stytch verifies it and redirects the browser to your backend callback route carrying an **Intermediate Token**.
- **Step 4: Exchange & Provisioning** — Your backend exchanges the intermediate token with Stytch for a valid Member Session and Organization context, then provisions your local database tenant if it is a new sign-up.
- **Step 5: Session Issuance** — The backend wraps the session into a secure cookie and redirects the user to the frontend dashboard.

## 2. Token and Cookie Strategy

Handling tokens correctly is vital for blocking XSS and CSRF vectors:

- **Intermediate Tokens** — These are temporary, single-use tokens meant only for the callback phase. They should never touch frontend storage; the backend consumes them immediately during the exchange.
- **Session Tokens** — Once exchanged, Stytch provides a Session Token or Session JWT. Store this exclusively in an **HttpOnly, Secure, and SameSite=Lax** cookie set by your backend.
- **Why this matters** — Preventing client-side JavaScript access stops cross-site scripting (XSS) attacks from exfiltrating session credentials. SameSite=Lax provides balanced protection against cross-site request forgery while allowing standard top-level navigation redirects.

## 3. When and How Tenants and Users Are Created

In a B2B SaaS context, a sign-up usually creates both a new **Organization (Tenant)** and a **Member (User)**.

- **The Source of Truth** — Stytch maintains the authoritative identity mapping (Stytch Organization ID and Stytch Member ID). Your PostgreSQL database mirrors this relationship.
- **Creation Trigger Point** — Creation happens **synchronously inside the backend callback handler** during the intermediate token exchange.
- **The Provisioning Process**:
    1. The backend receives the Stytch Organization ID and User profile data from the token exchange.
    2. The backend opens a database transaction and queries your local `tenants` table using the Stytch Organization ID.
    3. If the tenant does not exist, it creates a new record for the Organization (generating your internal Tenant UUID) and provisions the User linked to it.
    4. If it already exists, it simply maps the user session to the existing tenant.

## 4. Potential Error Scenarios & Mitigations

| Scenario                                                                                                                                            | Mitigation                                                                                                                |
| --------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------- |
| **Expired or Reused Intermediate Tokens** — a user clicks an old link or refreshes the callback page, so the token is invalid                       | Gracefully catch the Stytch error and redirect the user back to the login page with a clear message to request a new link |
| **Database Provisioning Failures** — Stytch successfully authenticates the user, but the local database transaction fails while creating the tenant | Ensure database operations are wrapped in atomic transactions with proper error logging so orphaned state is avoided      |
| **Email Rate-Limiting / Abuse** — malicious actors spamming the sign-up route to exhaust email quotas                                               | Enforce strict IP- and email-based rate limiting on the initial magic link request endpoint                               |

## 5. The Architectural Starting Spot

With your infrastructure, Prometheus observability, and Redis cache layer fully set up, your ideal entry point for implementation is building the **Stytch B2B Client & Redis Session Store Module** within your Uber Fx dependency injection graph. This forms the foundation for both sign-up and subsequent request validation.

## 6. Immediate Execution Roadmap

- **Step 1: Fx Redis Module** — Register your existing Redis client inside an Fx module so it can be injected cleanly into your authentication and caching layers.
- **Step 2: Stytch Configuration** — Create a configuration provider that reads your Stytch Project ID and Secret, initializing the official Stytch B2B client as an Fx singleton.
- **Step 3: Magic Link Initiation Route** — Build the public endpoint (`POST /api/auth/magic-link/send`) to trigger Stytch email dispatches with built-in IP rate-limiting to protect against abuse.
- **Step 4: Callback & Exchange Handler** — Implement the callback endpoint (`GET /api/auth/callback`) to consume the intermediate token, execute the Stytch session exchange, provision local database records via `sqlc`, and issue the opaque cookie.
- **Step 5: Session Middleware with Redis** — Create the Fiber middleware that reads the opaque cookie, checks your Redis cache (falling back to Stytch API on a miss), and populates `c.locals` with the verified tenant and user IDs.

## 7. Core Guardrails to Watch

- Enforce `HttpOnly`, `Secure`, and `SameSite=Lax` parameters on the cookie during Step 4.
- Wrap all local user/tenant database operations in atomic transactions using your `pgx/v5` pool.
