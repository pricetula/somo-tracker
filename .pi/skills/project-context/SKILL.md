---
name: project-context
description: Loads project-specific agent context from AGENTS.md files and skills. The single entry point for understanding how this project should be built.

# Project Context Loader

When Pi starts this project, it MUST first read the following files:

## Root Contract (root AGENTS.md)
- Architectural constraints
- Error handling rules
- Version policy
- Page navigation rules
- Layer isolation rules

## Layer Contracts
- **backend/AGENTS.md** – Backend handler/service separation, academic period resolver, migration testing, swagger verification
- **frontend/AGENTS.md** – Feature module architecture, proxy.ts migration, form patterns, shadcn purity, error handling
- **public/AGENTS.md** – Astro dev commands, documentation conventions

## Knowledge Base (docs/)
- **docs/features/** – Formal feature specs (FEATURE-*.md)
- **docs/gaps/** – Known missing work (GAP-*.md)  
- **docs/adr/** – Architecture decisions (ADR-*.md)
- **docs/planning/** – Pre-spec analysis, roadmaps, checklists
- **docs/templates/** – Gap, feature-spec, ADR templates

## Procedural Skills (.pi/skills/)
- **craft-docs** – Creates MDX docs from engineering prompts
- **generate-api** – Scaffolds Fiber API endpoints (service + router + Postman)
- **pure-shadcn** – Generates minimalist UI components
- **security-audit** – OWASP API security review workflow
- **migration-testing** – Migration integration test creation
- **error-contract** – Canonical error response enforcement
- **feature-module** – Feature-module architecture validation
- **agendas-compliance** – Cross-layer contract compliance audit

## Task Dispatch Rules
1. **If user asks to build something:**
   - Read the appropriate AGENTS.md file first
   - Check docs/features/ for existing specs
   - Load the matching skill if applicable
   - Follow the skill's workflow, not just its prose

2. **If user asks to review changes:**
   - Run the `/review` prompt template
   - Check all four layers' AGENTS.md files
   - Validate via relevant skills

3. **If user asks about architecture:**
   - Read root + layer AGENTS.md files
   - Review ADRs in docs/adr/
   - Check feature modules in frontend/src/features/
   - Check service interfaces in backend/internal/services/

4. **If user asks for feature implementation:**
   - Read frontend/AGENTS.md (frontend layer)
   - Check docs/features/ for related specs
   - Create the feature in a self-contained module
   - Document with docs/features/FEATURE-*.md

5. **Security check on any PR:**
   - Always load the `security-audit` skill
   - Refer to OWASP API Top 10 reference
   - Verify authorization logic, rate limiting, input validation

## Command Mapping
| Intent | Command | Skill |
|--------|---------|-------|
| Review changes | /review | agendas-compliance |
| Add endpoint | generate-api | generate-api |
| Make UI component | pure-shadcn | pure-shadcn |
| Create doc | craft-docs | craft-docs |
| Security audit | /security | security-audit |
| Migration test | /migration | migration-testing |
| Error validation | /error | error-contract |
| Feature check | /validate-feature | feature-module |

## Prerequisites
- All AGENTS.md files must be up to date
- All docs/ must have corresponding FEATURE-*.md or GAP-*.md entries
- All skills must have SKILL.md frontmatter
- All build commands must be executable
