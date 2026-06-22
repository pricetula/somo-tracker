---
name: pure-shadcn
description: Generates or refactors Next.js views using unadulterated, minimalist Shadcn components without custom Tailwind colors or nested div wrappers. Use this whenever the user asks for new UI, components, or layout updates.
---

# Pure Shadcn Engineering Procedure

You must follow these strict rules to enforce the team's visual "taste" and avoid structural bloat.

## 1. Absolute Token Enforcement (No Arbitrary Styles)

You are strictly forbidden from choosing hardcoded hex values, RGB strings, or standard tailwind color weights (like bg-slate-100, text-[#222], or text-blue-600).

The theme handles the styling. You must exclusively map elements to semantic CSS variables:

Layout backgrounds: bg-background or bg-muted/30

Text typography: text-foreground or text-muted-foreground

Accent focus points: bg-primary, text-primary-foreground

Exception: Standard operational system indicators only (text-destructive, text-emerald-600).

## 2. Flat Layout Tree (Anti-Div Soup)

Do not wrap elements or Shadcn primitives inside a div layout block unless introducing an active CSS Grid, Flex container, or concrete layout padding step.

Favor React Fragments over semantic-less structural divs when returning layout siblings.

Maximize Shadcn compound component properties to dictate alignment rather than throwing custom wrapper nodes around them.

## 3. Visual Layout Reference

No Cards or Borders: Do not use borders (border, border-input) or heavy card backgrounds (bg-card, shadow) to isolate dashboard sections.

Whitespace Separation: Use spatial layout gaps (space-y-6, gap-4) and clear typography hierarchy (font-medium text-muted-foreground) to define the structure.

Code Blueprint for the Agent
BAD Implementation (Bloated DOM, breaking the global theme config):

<div className="p-4 bg-[#fff] border border-gray-100 rounded-lg">
  <div className="flex items-center">
    <div className="text-sm font-semibold text-slate-800">
      <span className="text-blue-500">Active</span> Assessment
    </div>
  </div>
</div>

GOOD Implementation (Flat hierarchy, 100% theme variable driven):

<div className="p-4 bg-background">
  <p className="text-sm font-medium text-foreground">
    <span className="text-primary">Active</span> Assessment
  </p>
</div>
