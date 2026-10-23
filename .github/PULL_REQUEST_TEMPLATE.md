# Pull Request

## Summary
Brief description of the change and why it’s needed.

## Links
- Feature spec: `docs/features/FEATURE-*.md`
- Gap / ADR: `docs/gaps/GAP-*.md` or `docs/adr/ADR-*.md`
- Related issue: 

## Checklist
- [ ] `/review` passed — contracts validated across backend/frontend/docs
- [ ] `make compliance` passes locally
- [ ] Tests added/updated; migrations have `TestMigrator_*` integration test
- [ ] Error contract validated: `backend/internal/api/errors.go` → `frontend/src/lib/api/client.ts`
- [ ] Security audit completed for API changes — OWASP Top 10, BOLA/BFLA
- [ ] Docs/tooltips synced; `pnpm run audit:docs` clean
- [ ] No dumb back buttons, pure-shadcn rules respected
- [ ] Conventional commit messages used

## Testing
How was this tested locally?

## Screenshots / Notes
