# Implied and unmet requirements (gaps)

Gaps between documented requirements (`AGENTS.md` rules) and current implementation. These are **observed facts**, not inventions. Each needs a specification before work begins.

| ID (implied) | Gap | Evidence | Required before |
|--------------|-----|----------|-----------------|
| BE-020 / BE-021 | `platform/authz` permission matrix not implemented: `Require` always returns `FORBIDDEN "authorization check not implemented"`; no permission matrix data | `backend/internal/platform/authz/authz.go` | Any protected-domain specification |
| — | No protected domain handlers registered | `backend/internal/app/app.go` TODO comments: "Register protected domain handlers here" | Protected-domain specification (deferred; public routes only for now) |
| — | No finance/RBAC routes | `app.go` commented-out finance example | Finance specification (deferred) |
| — | No asynq worker handlers | `Worker.Start` is a `<-ctx.Done()` stub | Background-job specification (deferred) |
| — | Migrations run on app startup | `app.go` runs `migrator.Up` on start; note says production should run migrations as separate deploy step | Deployment specification |

**Resolved (no longer gaps):**

- Guardian is in MVP scope (previously listed as Phase 2 only per BE-035). Guardian role and its permission entries belong in the authz specification.
- CBC terminology: the `subjects`/`topics`/`subtopics` hierarchy (migration 05) is adopted; supports multiple countries (Kenya CBC, Tanzania, Uganda) via `countries` → `education_systems`. See `docs/product/README.md`.

## Open questions from the code

1. **Stytch organization → tenant/school mapping:** not fully spelled out in code; confirm before organization-selection workflows.
