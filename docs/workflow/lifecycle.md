# Development Lifecycle

Cross-layer flow. This flow is the ONLY sanctioned exception to changes staying within one layer.

1. Human writes/approves feature spec → status → approved
2. Update docs/api/openapi.yaml
3. Backend implementation + tests citing AC IDs
4. Regenerate API types → `make generate` must be idempotent
5. Frontend implementation + tests citing AC IDs
6. Run checks → `make compliance`, `/review`
7. Status → implemented by agent; → verified by human

Just-in-time speccing: spec at most 1-2 slices ahead of code.

Spec revision: if implementation reveals spec is wrong, update spec in same commit as code change.
