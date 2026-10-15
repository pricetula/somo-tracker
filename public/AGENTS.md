# Somotracker Public Site — Agent Architecture Contract

Astro — marketing website.

---

## 1. Stack

- **Framework:** Astro (latest)
- **UI components:** Astro components + Tailwind CSS
- **Styling:** Tailwind CSS (`tailwind.config.mjs`)

---

## 2. Directory Layout

```
src/
├── layouts/                    # BaseLayout.astro, etc.
├── pages/                      # File-based routing, e.g. index.astro, pricing.astro
│   └── [slug]/
│       └── index.astro         # Dynamic marketing pages
├── components/                 # Reusable Astro components
│   └── marketing/              # Site-specific components (Hero, Navbar, Footer, CTA)
├── styles/
│   └── global.css              # Tailwind base + global styles
└── lib/
    └── utils.ts                # cn() helper and global utilities

public/                         # Static assets, copied as-is to dist/
```

---

## 2. Package Manager

Use **pnpm** exclusively — never `npm` or `yarn`.

- `pnpm install` — local dev
- `pnpm install --frozen-lockfile` — CI/Docker
- `pnpm add <pkg>` / `pnpm add -D <pkg>` / `pnpm remove <pkg>`
- `pnpm exec` / `pnpm dlx` for one-off commands (never global installs)
- Use `--ignore-scripts` in CI/Docker unless a postinstall script is explicitly required.

---

## 3. Component Rules

- Use `.astro` files for pages and components. Prefer zero-JS output.
- Use Islands architecture for interactive parts only.
- Marketing components in `src/components/marketing/` are Astro components.
- Keep `pages/*.astro` files lean — compose components, no inline business logic.

---

## 4. Rendering Strategy

- All marketing pages are prerendered statically by default.
- Use `export const prerender = true` explicitly where needed.
- Avoid client-side data fetching; fetch at build time via `getStaticPaths` when dynamic routes are required.
- Islands are opt-in for interactivity only.

---

## 5. Styling

- Tailwind utility classes only. No custom CSS outside of `src/styles/global.css`.
- Theming is controlled via CSS variables in `global.css`.
- Use `cn()` from `$lib/utils` for conditional class merging.
