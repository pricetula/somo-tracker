# Somotracker

School management system.

## Workspace

- `frontend/` – TanStack Start + Tailwind + shadcn/ui
- `backend/` – Go Fiber v3 API + worker, PostgreSQL, Redis, Asynq
- Monorepo layout per BE-001

## Local development

Requires Doppler for secrets. See docs.

```bash
docker compose up -d postgres redis
doppler run -- make dev
```

Do not commit secrets. No `.env` files in repo (BE-033).
