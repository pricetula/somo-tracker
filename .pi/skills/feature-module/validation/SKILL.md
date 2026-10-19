---
name: feature-module-validation
description: Validate feature modules adhere to the feature-module architecture (isolated logic, UI, services, types).

---

# Feature Module Validation

## When to Run
This skill is triggered when:
- A new feature is created under `frontend/src/features/<name>/`
- A feature is modified after initial scaffolding
- Before merging a feature branch

## Validation Rules

### 1. Directory Structure
Each feature must have exactly these top-level files:
```
features/<name>/
├── index.ts          # Public API entry point
├── components/       # Presentational UI (shadcn primitives only)
├── hooks/            # Data fetching and local state
├── services/         # API clients, server actions, SDK wrappers
├── types/            # TypeScript interfaces for this feature
└── index.ts          # Re-exports (single import entry point)
```

### 2. Import Rules
- **Cross-feature imports** are allowed for now but discouraged
- **Shared logic** that is truly generic belongs in `lib/`
- **No circular dependencies** between features

### 3. Component Rules (Pure Shadcn)
- All UI components live in `features/<name>/components/`
- No arbitrary `<div>` wrappers around Shadcn primitives
- Components use `render={<Component />}` not `asChild`
- No custom Tailwind colors or hex values
- Each component is a single file (no multi-component files)

### 4. Service Layer
- All business logic lives in `services/`
- Services call sqlc-generated queries only (`*sqlc.Queries`)
- No direct `pgxpool` or raw SQL in handlers
- `internal/services.<domain>.Service` pattern recommended

### 5. Type Safety
- `types/` must export TypeScript interfaces
- `index.ts` must export a single `export` statement (no named exports)
- All types must be usable by other features (shared types go to `lib/`)

## Integration Points
- Triggered by `/validate-feature` command
- Uses `frontend/src/features/<name>/index.ts` as the primary entry point
- Runs `pnpm lint` before marking feature ready
- Can combine with `generate-api` skill for service scaffolding

## Prerequisites
- `frontend/src/features/<name>/index.ts` (exists)
- `frontend/src/features/<name>/types/` (exists)
- `frontend/src/features/<name>/services/` (exists)
- `frontend/src/features/<name>/components/` (exists)
- `lib/` with shared types (exists)
