# Public / Marketing Agent Rules

## Architecture
- **Astro static site** for marketing pages.
- Deployed separately from the main application.
- No authentication or session handling.

## Testing
- Lint: `cd marketing && pnpm lint`
- Typecheck: `cd marketing && pnpm typecheck`

## Rules
- Keep marketing site lightweight; no heavy client-side frameworks.
- All images optimized; favicon and OG images in `public/`.
