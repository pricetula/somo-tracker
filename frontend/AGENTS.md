# Somotracker Frontend — Agent Rules

Version: 0.1.0
Owner: Platform team

## Rule list

### FE-001 Feature modules
Feature modules are self-contained. External imports only via `features/<name>/index.ts`. No business logic in `app/`.
Enforced by: TODO(tooling) – suggested: ESLint boundaries/no-restricted-imports

### FE-002 No back navigation
No back buttons or "Back to" links in pages or feature containers. Navigation belongs to app shell.
Enforced by: TODO(tooling) – suggested: lint for ArrowLeft / "Back to" strings

### FE-003 Proxy
`src/proxy.ts` replaces middleware; default export only; matcher required.
Enforced by: manual review

### FE-004 No setState in effect
Never call `setState` inside `useEffect`. Use `useMemo` or event handlers.
Enforced by: TODO(tooling) – suggested: eslint-plugin-react-hooks

### FE-005 API client
All API calls via `src/lib/api/client.ts`. Paths start with `/api/`. Never hardcode `/backend`.
Enforced by: TODO(tooling) – suggested: lint for fetch('/backend'

### FE-006 ApiError contract
`ApiError` with `status/code/message/errors`. Mutations require `onError` toast.
Enforced by: manual review

### FE-007 Forms
Forms use shadcn `Form` + `zodResolver` + `useForm`. No raw `<form>` + `useState`.
Enforced by: manual review

### FE-008 Shadcn UI
Never hand-edit `src/components/ui/`. Add via `pnpm dlx shadcn@latest add`.
Enforced by: manual review

### FE-009 Dates
Dates via `date-fns` only.
Enforced by: TODO(tooling) – suggested: lint for Intl.DateTimeFormat

### FE-010 API types
API types generated from OpenAPI spec. Never hand-write them.
Enforced by: TODO(tooling) – suggested: `make generate` idempotency check

## General
See root AGENTS.md for commit policy and document-led workflow.
Library versions: see package.json.
