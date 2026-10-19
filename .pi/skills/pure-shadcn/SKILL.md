---
name: pure-shadcn
description: Generates or refactors Next.js views using unadulterated, minimalist Shadcn components without custom Tailwind colors or nested div wrappers. Use whenever the user asks for new UI, components, or layout updates.

---

# Pure Shadcn Engineering Procedure

## Absolute Token Enforcement (No Arbitrary Styles)
You are strictly forbidden from choosing hardcoded hex values, RGB strings, or standard Tailwind color weights (like bg-slate-100, text-[#222], or text-blue-600).

The theme handles the styling. You must exclusively map elements to semantic CSS variables:

- **Layout backgrounds**: `bg-background` or `bg-muted/30`
- **Text typography**: `text-foreground` or `text-muted-foreground`
- **Accent focus points**: `bg-primary`, `text-primary-foreground`
- **Exception**: Standard operational system indicators only (`text-destructive`, `text-emerald-600`)

## Flat Layout Tree (Anti-Div Soup)
Do not wrap elements or Shadcn primitives inside a div layout block unless introducing an active CSS Grid, Flex container, or concrete layout padding step.

- Favor React Fragments over semantic-less structural divs when returning layout siblings
- Maximize Shadcn compound component properties to dictate alignment rather than throwing custom wrapper nodes around them
- Use `space-y-*` and `gap-*` for spacing instead of padding/margin on wrapper divs

## Visual Layout Reference
**No Cards or Borders**: Do not use borders (`border`, `border-input`) or heavy card backgrounds (`bg-card`, `shadow`) to isolate dashboard sections.

**Whitespace Separation**: Use spatial layout gaps (`space-y-6`, `gap-4`) and clear typography hierarchy (`font-medium text-muted-foreground`) to define the structure.

**Code Blueprint**:

BAD Implementation (Bloated DOM, breaking the global theme config):
```tsx
<div className="p-4 bg-[#fff] border border-gray-100 rounded-lg">
  <div className="flex items-center">
    <div className=" font-semibold text-slate-800">
      <span className="text-blue-500">Active</span> Assessment
    </div>
  </div>
</div>
```

GOOD Implementation (Flat hierarchy, 100% theme variable driven):
```tsx
<section className="space-y-4">
  <h2 className="font-medium text-muted-foreground">
    <span className="text-primary">Active</span> Assessment
  </h2>
  {/* content */}
</section>
```

## Shadcn Component Rules
- Files under `src/components/ui/` are auto-generated shadcn primitives
- The only way to add or change one is `pnpm dlx shadcn@latest add <component>`
- Bugs, type errors, or Tailwind warnings in these files are fixed by re-adding or upgrading via the shadcn CLI — never by hand
- If a shadcn component has a type mismatch with its underlying library (e.g. `react-day-picker`), update the library or re-add the component
- **Base UI render prop**: Base UI primitives in the current shadcn set (DropdownMenu, SidebarMenuButton, etc.) use a `render` prop for composition, not the legacy `asChild` pattern. Always use `render={<Component />}` instead of `asChild`. Do not use `asChild`.

## Page Layout Rules
- Use vertical stacking of sections with `space-y-*` gap to organise page content
- Do not use `<Tabs>` or tab navigation components to split page content
- Every section gets an `<h2>` / `<h3>` heading and stacks naturally down the page
- Sub-navigation belongs in the sidebar or in dedicated feature-level route pages

## Integration Points
- Uses `frontend/AGENTS.md` for page layout and component rules
- Uses `frontend/src/components/ui/` for shadcn primitives
- Uses `frontend/src/features/<name>/components/` for feature components
- Triggers `pnpm lint` before marking UI ready
- Can combine with `feature-module-validation` skill for feature scaffolding

## Prerequisites
- `frontend/src/components/ui/` directory exists with shadcn primitives
- `frontend/src/features/<name>/components/` directory exists
- `frontend/AGENTS.md` exists with component rules
- `pnpm lint` is executable
- Tailwind CSS is configured with semantic variables