# Frontend Agent Rules

## Architecture
- **Next.js App Router** with React 19, TypeScript, Tailwind CSS, and shadcn/ui.
- **BFF pattern**: Frontend API routes under `src/app/api/` proxy to the Go backend. The Go backend sets session cookies directly on responses forwarded through Next.js.
- **Component structure**: One component per file, flat layout tree, no div soup. Shadcn components are used untouched (no manual modifications or custom Tailwind colors).
- **Error handling**: All async handlers have try/catch. Error states render user-facing messages. 403 and 502 responses from Turnstile verification show specific messages.

## shadcn/ui Components

### Installation
- All shadcn primitives live under `src/components/ui/` and are added **only** via the shadcn CLI:
  ```
  pnpm dlx shadcn@latest add <component>
  ```
- Never hand-edit files in `src/components/ui/`. Bugs, type errors, or Tailwind warnings are fixed by re-adding or upgrading via the shadcn CLI.
- If a shadcn component has a type mismatch with its underlying library (e.g. `react-day-picker`), update the library or re-add the component — never patch the generated file manually.

### Styling Rules (Absolute Token Enforcement)
- **Never** use hardcoded hex values, RGB strings, or arbitrary Tailwind color weights (e.g. `bg-[#fff]`, `text-[#222]`, `bg-slate-100`, `text-blue-600`).
- Map all styling exclusively to semantic CSS variables:
  - Layout backgrounds: `bg-background`, `bg-muted`, `bg-muted/30`
  - Text: `text-foreground`, `text-muted-foreground`
  - Accents: `bg-primary`, `text-primary-foreground`
  - System indicators only: `text-destructive`, `text-emerald-600`
- No custom Tailwind colors. No hardcoded hex colors anywhere.

### Flat Layout Tree (Anti-Div Soup)
- Do not wrap elements or shadcn primitives inside a `div` unless introducing an active CSS Grid, Flex container, or concrete layout padding step.
- Favor React Fragments over semantic-less structural divs when returning layout siblings.
- Maximize shadcn compound component properties (e.g. `Dialog.Header`, `Dialog.Content`, `Form.Field`) rather than custom wrapper nodes.
- Use `space-y-*` and `gap-*` for spacing instead of `p-*`/`m-*` on wrapper divs.

### Visual Layout Reference
- **No cards or borders**: Do not use `border`, `border-input`, `bg-card`, or `shadow` to isolate page sections.
- **Whitespace separation**: Use spatial layout gaps (`space-y-6`, `gap-4`) and typography hierarchy (`font-medium text-muted-foreground`) to define structure.

### Base UI Render Prop Pattern
- Base UI primitives (DropdownMenu, SidebarMenuButton, etc.) use a `render` prop, not `asChild`.
- Always use `render={<Component />}` instead of `asChild`.
- Never use `asChild`.

### Page Layout Rules
- Stack sections vertically with `space-y-*` gaps to organise page content.
- Do not use `<Tabs>` or tab navigation to split page content.
- Every section gets an `<h2>` or `<h3>` heading and stacks naturally down the page.
- Sub-navigation belongs in the sidebar or dedicated feature-level route pages.

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
- Form implementations follow the onboarding-form pattern (react-hook-form + zod).
- Icon library: lucide-react.
- Follow git conventions in root `AGENTS.md` (conventional commits, pre-push compliance, no secrets in git).
