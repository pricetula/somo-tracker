# Frontend Agent Rules

## Architecture
- **Next.js App Router** with React 19, TypeScript, Tailwind CSS, and shadcn/ui.
- **BFF pattern**: Frontend API routes under `src/app/api/` proxy to the Go backend. The Go backend sets session cookies directly on responses forwarded through Next.js.
- **Component structure**: One component per file, flat layout tree, no div soup. Shadcn components are used untouched (no manual modifications or custom Tailwind colors).
- **Error handling**: All async handlers have try/catch. Error states render user-facing messages. 403 and 502 responses from Turnstile verification show specific messages.

## Turnstile
- Site key `0x4AAAAAAGHleNH8xjjhEUQL` is hard-coded in `src/components/turnstile.tsx` (no `NEXT_PUBLIC_TURNSTILE_SITE_KEY` env var).
- Token kept in state; submit button disabled until token exists.
- Token cleared on widget expiry/error; widget reset and token cleared after every submit attempt.
- Token sent to backend as `turnstile_token` alongside `email`.

## Required Configuration
| Variable | Required | Description |
|----------|----------|-------------|
| `API_URL` | Yes | Backend API URL |
| `APP_URL` | Yes | Public frontend URL |
| `SESSION_COOKIE_NAME` | Yes | Must match backend |
| `APP_ENV` | Yes | `development` or `production` |
| `NODE_ENV` | Yes | Node environment |
| `PORT` | Yes | Frontend port (3000) |
| `HOSTNAME` | Yes | Hostname for Next.js |
| `TURNSTILE_SITE_KEY` | If enabled | Turnstile site key for client-side widget |

## Testing
- Typecheck: `cd frontend && pnpm typecheck`
- Lint: `cd frontend && pnpm lint`
- Tests: `cd frontend && pnpm test`

## Rules
- No "dumb back buttons" (no ArrowLeft icons in features/pages).
- No hardcoded hex colors (use semantic CSS vars).
- All error boundaries wrap major routes.
- API mutations pass IDs in request body (not just URL params).
- Form implementations follow the onboarding-form pattern.
- Follow git conventions in root `AGENTS.md` (conventional commits, pre-push compliance, no secrets in git).
