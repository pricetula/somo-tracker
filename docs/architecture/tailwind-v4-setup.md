# Tailwind v4 + Vite Setup

## Why styles disappear after shadcn install

Tailwind CSS v4 is not PostCSS-based. With Vite it **must** be compiled by the `@tailwindcss/vite` plugin.

Without the plugin Vite treats `src/styles/globals.css` as plain CSS:
* `@import "tailwindcss"` is not expanded
* `@theme`, `@utility`, `@custom-variant` at-rules from `tailwindcss`, `tw-animate-css` and `shadcn/tailwind.css` are never processed
* No CSS variables / utilities are emitted → shadcn components render unstyled

## Correct setup

frontend/package.json
```json
"devDependencies": {
  "@tailwindcss/vite": "^4.3.3",
  "tailwindcss": "^4.3.3",
  ...
}
```

frontend/vite.config.ts
```ts
import tailwindcss from '@tailwindcss/vite';

export default defineConfig({
  plugins: [
    tanstackStart(),
    tailwindcss(),   // <-- required for Tailwind v4
    react(),
  ],
  ...
});
```

frontend/src/styles/globals.css – first line must be Tailwind import
```css
@import "tailwindcss";
@import "tw-animate-css";
@import "shadcn/tailwind.css";
@import "@fontsource-variable/geist";
@import "@fontsource-variable/inter";
...
```

components.json
* `tailwind.css` points to `src/styles/globals.css`
* `style` = `base-mira` or other shadcn v4 style – no `tailwind.config.js` needed

Restart `pnpm dev` after adding the plugin.

## Checklist

- [ ] `@tailwindcss/vite` installed as devDependency
- [ ] `tailwindcss()` added to Vite plugins before `react()`
- [ ] `globals.css` imported once, e.g. in `src/entry.client.tsx`
- [ ] No PostCSS `tailwind.config.js` left over from v3
- [ ] `pnpm dev` restarted after change

See also: frontend/AGENTS.md FE-016 semantic tokens, ADR-0004 TanStack Start migration.
