# Features Index

This index lists all documented features for Somotracker. Each feature lives under `docs/features/FEATURE-<slug>.md` using the template in `docs/templates/feature-spec.md`.

## Legend
- Status: Draft | Proposed | Accepted | In Progress | Done | Deprecated

## Features

| Feature | Slug | Status | Owner | Layer |
|---------|------|--------|-------|-------|
| Backend Schema Overview | backend-schema | Done | Platform | backend |
| Bulk Operations Framework | bulk-jobs | Done | Platform | backend |
| Enhanced Student Profile | enhanced-student-profile | Proposed | Product / Platform | backend, frontend |
| Enhanced Student Listing | enhanced-student-listing | Proposed | Product / Platform | backend, frontend |
| ... | ... | ... | ... | ... |

## How to add a feature
1. Copy `docs/templates/feature-spec.md` to `docs/features/FEATURE-<slug>.md`
2. Fill in Problem, Solution, Acceptance Criteria
3. Link related GitHub Issues
4. Update this index

## References
- `docs/backend_schema.md` - Current database schema summary
- Monorepo structure: backend/, frontend/, public/, docs/
