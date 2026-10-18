---
name: security-audit
description: Audit and harden REST/GraphQL/gRPC APIs for OWASP Top 10 vulnerabilities. Used before PR approval, during design, and for production deployment.

# Security Audit Workflow

## When to Run
This skill is triggered when:
- A PR touches API endpoints, auth middleware, or API gateway config
- User requests an "API security review", "pentest prep", "API audit", or similar
- Designing a new endpoint before implementation

## Step 1 — Threat Model the Endpoint
For each endpoint/resource in scope, answer:
1. **Object-level**: What entity is read/written, and by what ID?
2. **Authorization**: Who is supposed to call it, with what role?
3. **Worst-case**: What can an authenticated-but-unauthorized user do?
4. **External calls**: Does this endpoint call another service based on user input?
5. **Sensitive flows**: Is this a checkout/transfer/password-reset that could be abused?

## Step 2 — OWASP API Top 10 Checklist
Walk through `reference/owasp-api-top10-2023.md` category by category:

| Category | Control to Verify |
|----------|-------------------|
| **API1 — BOLA** | Every object-fetching endpoint filters by `owner_id`/scope, not just `is_logged_in` |
| **API3 — BPLA** | Serializers respect field-level authorization (no mass assignment, no over-fetching) |
| **API5 — BFLA** | Admin-only ops gated at handler level, not hidden from UI |
| **API4 — Unrestricted Consumption** | Rate limits, pagination caps, payload size limits on expensive operations |
| **API6 — Unrestricted Flow** | Can this flow be automated/scripted at scale (scalping, spam)? |
| **API7 — SSRF** | Any user-input URL/webhook is allowlisted, not denylisted |
| **API2 — Auth** | Proper auth flow (JWT/CSRF/RLS) for every endpoint |
| **API8 — Misconfig** | Security headers, CSP, HSTS present |
| **API9 — Inventory** | Endpoints documented in OpenAPI/schema |
| **API10 — Third-party** | External API calls validated for secrets/permissions |

## Step 3 — Verify Runtime Behavior
- **Authorization tests**: For every object-scoped endpoint, assert user A cannot read/write user B's object by ID substitution
- **Schema/contract testing**: Use Schemathesis or similar to fuzz inputs against OpenAPI spec
- **Dynamic scanning**: Run OWASP ZAP against a running instance for misconfigurations
- **API inventory**: Confirm all endpoints are documented in the OpenAPI/GraphQL schema

## Step 4 — Reporting
Structure findings as:
- **Critical** – BOLA, BFLA, SSRF, unrestricted flow
- **High** – Missing rate limits, missing auth on sensitive flows
- **Medium** – Missing security headers, incomplete input validation
- **Low** – Documentation gaps, minor misconfigurations

## Integration Points
- Triggered by `/security` command
- Uses `pi.registerTool()` for security scanning tools (OWASP ZAP, etc.)
- Outputs findings to `security-report.md`
- Can be combined with `generate-api` skill for new endpoint hardening

## Prerequisites
- `reference/owasp-api-top10-2023.md` (exists in project)
- `reference/review-checklist.md` (exists in project)
- Access to running API instance for dynamic scanning
