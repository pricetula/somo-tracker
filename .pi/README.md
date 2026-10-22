# Pi Agent Configuration

This directory configures the pi coding agent's behavior for this project.

## Structure
```
.pi/
├── README.md              # This file
├── prompts/                # Reusable command templates
│   └── review.md           # /review command template
└── skills/                # Specialized task procedures
    ├── craft-docs/          # MDX document creation
    ├── generate-api/        # Fiber API endpoint scaffolding
    ├── pure-shadcn/         # Minimalist UI component generation
    ├── security-audit/      # OWASP API security review
    ├── migration-testing/   # PostgreSQL migration integration tests
    ├── error-contract/      # Error response contract enforcement
    ├── feature-module/      # Feature-module architecture validation
    └── agendas-compliance/  # Cross-layer AGENTS.md compliance audit
```

## How Skills Work
- Each skill is auto-discovered by pi at startup
- Skill names appear in the model's context as routing hints
- Use `/skill:name` to force-load a specific skill
- Each skill's `SKILL.md` defines procedural instructions and integration points

## How Prompts Work
- Prompt files become `/` commands in the editor
- Examples: `/review <area>`, `/skill:name args`
- Run `/reload` after changes during active session

## Integration with AGENTS.md
- Root AGENTS.md = architectural constraints (the "constitution")
- Layer AGENTS.md files = layer contracts
- Skills = procedural workflows (how to do things)
- Prompts = reusable command templates
- Docs = knowledge base (spec, requirements)

All four layers work together to make every code change compliant with project standards.