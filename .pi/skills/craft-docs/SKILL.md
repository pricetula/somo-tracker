---
name: craft-docs
description: Crafts and structures MDX files under content/docs/ from raw engineering prompts, ensuring standard frontmatter compliance and triggering sync audits. Use when user requests "add documentation page" or "update existing doc".

---

# Craft Documentation

## Output Example
```
---
title: "New Feature Overview"
description: "Brief description of the new feature."
tooltipSummary: "Overview of new feature"
---

# New Feature Overview

## Context {#context}
...
```

## Task Loop

### Step 1 — Parse User Input to Extract Feature Scope
1. Identify the feature name and purpose
2. Determine the target audience (school admin, teacher, student, etc.)
3. Extract any specific requirements or constraints
4. Check for existing related features in `docs/features/`

### Step 2 — Draft a Fresh File Under `content/docs/[slug].mdx`
- Use the template from `docs/templates/feature-spec.md`
- Include `title`, `description`, and `tooltipSummary` in frontmatter
- `tooltipSummary` must be plain text, under 160 characters
- Use `## Section {#anchor}` format for headings (anchor = kebab-case)

### Step 3 — Scaffold Headings with Structural Anchor References
- Use `## Context {#context}` for background
- Use `## Implementation {#implementation}` for technical details
- Use `## Testing {#testing}` for verification steps
- Use `## Edge Cases {#edge-cases}` for known limitations

### Step 4 — Output the Precise JSX Line for FeatureHelp
```
<FeatureHelp slug="[slug]" anchorId="[anchor]" />
```
- Replace `[slug]` with the filename (without .mdx)
- Replace `[anchor]` with the kebab-case heading

### Step 5 — Run Verification Pipelines
- Run `pnpm run docs:lint` to validate frontmatter
- Run `pnpm run audit:docs` to check tooltip sync
- Run `pnpm run docs:build` to generate MDX

## Integration Points
- Uses `docs/templates/feature-spec.md` as template
- Uses `docs/templates/gap.md` for gap documentation
- Uses `docs/templates/adr.md` for architecture decisions
- Triggers `pnpm run audit:docs` for tooltip synchronization
- Generates `content/docs/[slug].mdx` files

## Prerequisites
- `docs/templates/` directory exists with templates
- `content/docs/` directory exists
- `pnpm run docs:lint` is executable
- `pnpm run audit:docs` is executable
- `pnpm run docs:build` is executable