# Somotracker Public Site — Agent Architecture Contract

SvelteKit + shadcn-svelte — marketing website.

---

## 1. Stack

- **Framework:** SvelteKit (latest)
- **UI components:** shadcn-svelte
- **Styling:** Tailwind CSS (`tailwind.config.ts`)

---

## 2. Directory Layout

```
src/
├── routes/                     # SvelteKit file-based routing
│   ├── +layout.svelte
│   ├── +page.svelte            # Home / landing page
│   └── [slug]/
│       └── +page.svelte        # Dynamic marketing pages (/pricing, /about, …)
│
├── lib/
│   ├── components/
│   │   ├── ui/                 # shadcn-svelte primitives (do not hand-edit)
│   │   └── marketing/          # Site-specific components (Hero, Navbar, Footer, CTA)
│   ├── utils.ts                # cn() helper and global utilities
│   └── types.ts                # Shared TypeScript types
│
└── app.css                     # Tailwind base + shadcn CSS variable definitions
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

- Add shadcn-svelte components with `pnpm dlx shadcn-svelte@latest add <component>`. Never hand-edit files in `src/lib/components/ui/`.
- Marketing components in `src/lib/components/marketing/` wrap or compose shadcn primitives.
- Keep `+page.svelte` files lean — they compose components, no inline logic or styles.

---

## 4. Rendering Strategy

- Prefer static prerendering (`export const prerender = true`) on all marketing pages.
- Use `+page.server.ts` only when server-side data fetching is required (CMS, feature flags).
- Avoid client-side data fetching on marketing pages — it hurts Core Web Vitals.

---

## 5. Styling

- Tailwind utility classes only. No custom CSS outside of `app.css`.
- shadcn-svelte theming is controlled via CSS variables in `app.css` — modify variables, not component files.
- Use `cn()` from `$lib/utils` for conditional class merging.
