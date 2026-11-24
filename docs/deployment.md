# Deployment Overview

## Production Endpoints

| Service | URL | Provider | Notes |
|---------|-----|----------|-------|
| Frontend (Next.js) | `https://app.somotracker.com` | Vercel | Connected to GitHub `main` — auto-deploys on push |
| Backend (Go API) | `https://api.somotracker.com` | Render | Deployed via Render service |
| Documentation | `https://docs.somotracker.com` | — | Static docs (if applicable) |

## Frontend — Vercel

- **App:** `app.somotracker.com`
- **Provider:** Vercel
- **Source:** GitHub repo `pricetula/somo-tracker`, directory `frontend/`
- **Deployment:** Automatic on push to `main` (Vercel GitHub integration)
- **Environment variables** (configured in Vercel dashboard):
  - `API_URL` → `https://api.somotracker.com`
  - `APP_URL` → `https://app.somotracker.com`
  - `SESSION_COOKIE_NAME` → `__Host-session`
  - `APP_ENV` → `production`
  - `NODE_ENV` → `production`
  - `PORT` → `3000`
  - `HOSTNAME` → `0.0.0.0`

## Backend — Render

- **App:** `api.somotracker.com`
- **Provider:** Render
- **Source:** GitHub repo `pricetula/somo-tracker`, directory `backend/`
- **Deployment:** Render service connected to GitHub
- **Environment variables** (injected via Doppler at runtime):
  - `DATABASE_URL` → PostgreSQL (production)
  - `REDIS_URL` → Redis (production)
  - `BREVO_API_KEY`, `BREVO_SENDER_EMAIL`, `BREVO_SENDER_NAME`
  - `EMAIL_HMAC_KEY`
  - `SESSION_COOKIE_NAME` → `__Host-session`
  - `TURNSTILE_SECRET_KEY`
  - `TURNSTILE_EXPECTED_HOST` → `api.somotracker.com`
  - `CORS_ORIGINS` → `https://app.somotracker.com`
  - `APP_ENV` → `production`
  - `PORT` → `3030`

## CORS Configuration

The Go backend CORS is configured to allow only the frontend origin:
- `CORS_ORIGINS=https://app.somotracker.com`
- Credentials enabled (cookies forwarded cross-origin)

## Local Development vs Production

| | Local | Production |
|---|-------|------------|
| Frontend URL | `http://localhost:3000` | `https://app.somotracker.com` |
| Backend URL | `http://localhost:3030` | `https://api.somotracker.com` |
| Session cookie | `session` (insecure) | `__Host-session` (Secure, HTTPS only) |
| Secrets | Doppler (dev config) | Doppler (production config) |

## Secrets Management

All secrets are injected by **Doppler** at runtime. No `.env` values are committed to git.

- **Backend secrets:** Doppler → Render environment variables
- **Frontend secrets:** Vercel environment variables (non-sensitive config only; sensitive keys never exposed to browser)

## DNS

| Record | Type | Value |
|--------|------|-------|
| `app.somotracker.com` | CNAME / A | Vercel |
| `api.somotracker.com` | CNAME / A | Render |

## How Changes Deploy

1. Feature branch → PR against `develop` → merge to `develop`
2. Release PR: `develop` → `main`
3. Push to `main` triggers:
   - **Vercel:** Auto-deploys frontend to `app.somotracker.com`
   - **Render:** Auto-deploys backend to `api.somotracker.com`
4. GitHub CI runs tests on every push/PR to `main`
