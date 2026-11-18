<!-- BEGIN:nextjs-agent-rules -->
This is NOT the Next.js you know

This version has breaking changes — APIs, conventions, and file structure may all differ from your training data. Read the relevant guide in node_modules/next/dist/docs/ (resolved from this file's directory; in monorepos the next package may not be visible from the repo root) before writing any code. Heed deprecation notices.

This block is written and re-added by next dev — verify at node_modules/next/dist/server/lib/generate-agent-files.js. Removing it from a diff only re-creates the uncommitted change; committing it with your work keeps the tree clean.

<!-- END:nextjs-agent-rules -->

# Somotracker — Frontend Agent Contract

**Application:** `frontend/` — Next.js App Router + Tailwind + shadcn/ui.

## Technology stack

- **Framework:** Next.js lates (App Router)
- **Language:** TypeScript (strict)
- **Styling:** Tailwind CSS + shadcn/ui components
- **State:** React hooks + server actions where appropriate
- **Tests:** None configured yet (add vitest or @testing-library/react as needed)

## Backend-for-Frontend (BFF) cookie architecture

The frontend is a BFF that sits between the browser and the Go backend:

```
Browser (app.somotracker.com) ←→ Next.js API routes ←→ Go backend (api.somotracker.com)
```

### Cookie contract

All auth cookies are `HttpOnly` and set on the Next.js host. Next.js API routes proxy to Go and forward `Set-Cookie` headers from Go responses to the browser. On outgoing requests to Go, Next.js forwards the `Cookie` header.

| Cookie | Purpose | TTL | Set by |
|--------|---------|-----|--------|
| `__Host-session_token` / `session_token_development` | Auth session | 30 days | Go → Next.js → Browser |

| `__Host-onboarding` / `onboarding_development` | Onboarding ticket | 10 min | Go → Next.js → Browser |

### API routes (BFF proxy)

All under `app/api/`:

- `POST /api/auth/request-magic-link` → `POST /api/v1/auth/request-magic-link` on Go
- `POST /api/auth/authenticate-token` → `POST /api/v1/auth/authenticate-token` on Go (forwards Set-Cookie)
- `POST /api/auth/logout` → `POST /api/v1/auth/logout` on Go
- `GET /api/me` → `GET /api/v1/me` on Go (forwards session cookie)
- `GET/PUT /api/onboarding` → `GET/PUT /api/v1/onboarding/current` on Go
- `POST /api/onboarding/complete` → `POST /api/v1/onboarding/complete` on Go

### Cookie forwarding

`lib/go-client.ts` provides `goRequest()` which:
1. Reads `__Host-session_token` (or dev equivalent) from `cookies()`
2. Forwards it in the `Cookie` header to Go
3. Forwards `Set-Cookie` headers from Go responses to the browser
4. Forwards `Origin` header on mutations for Go CSRF middleware

## Routes

| Route | Purpose | Auth |
|-------|---------|------|
| `/login` | Magic link email entry | Public |
| `/auth/callback` | Stytch magic link redirect | Public |
| `/dashboard` | Protected dashboard | Session required |
| `/api/*` | BFF API routes | See above |

## Secrets

No `.env` files. Secrets are injected by Doppler at runtime via `doppler run --`. Required env vars:

- `PUBLIC_APP_URL` — frontend URL (e.g., `https://app.somotracker.com`)
- `PUBLIC_API_URL` — Go backend URL (e.g., `https://api.somotracker.com` or `http://backend:8080` in Docker)

See `.env.example` for local reference.

## Commands

| Purpose | Command |
|---------|---------|
| Dev server | `npm run dev` |
| Production build | `npm run build` |
| Production start | `npm run start` |
| Lint | `npm run lint` |
| Typecheck | `npm run typecheck` |
