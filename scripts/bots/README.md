# Bot Factory Scripts

Executable equivalents of `.pi/skills/` workflows.

## Scripts
- `error-contract.sh` – Validates canonical error response shape backend/frontend.
- `api-audit.sh` – Handler/service separation, swagger idempotency, router verification.
- `ui-audit.sh` – Pure-shadcn rules, no ArrowLeft, no hex colors.
- `security-audit.sh` – OWASP API Top 10 basics, BOLA/BFLA checks.
- `migration-testing.sh` – Ensures each migration has TestMigrator_*.
- `feature-module.sh` – Validates frontend feature module structure.
- `run-all.sh` – Runs all bots.

Run locally: `make bots` or `make compliance`.
CI runs via `.github/workflows/bots.yml`.
