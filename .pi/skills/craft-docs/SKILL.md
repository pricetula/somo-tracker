---
name: craft-docs
description: "Crafts and structures MDX files under content/docs/ from raw engineering prompts, ensuring standard frontmatter compliance and triggering sync audits."
---

# Skill Task Loop
1. Parse user input to extract feature scope.
2. Draft a fresh file under `content/docs/[slug].mdx` containing `title`, `description`, and a plain-text `tooltipSummary`.
3. Scaffold headings with structural anchor references (e.g., `## Context Allocation {#context-allocation}`).
4. Output the precise JSX line the user needs to invoke the tool: `<FeatureHelp slug="[slug]" anchorId="[anchor]" />`.
5. Run verification pipelines: `npm run audit:docs`.
