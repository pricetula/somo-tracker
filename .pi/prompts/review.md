---
description: Review changes against project contracts. Validates AGENTS.md compliance, security, migrations, and error handling.
argument-hint: "[area] - backend, frontend, security, migration, all"
---
Review the recent changes against the project's architectural contracts.

Focus on:
1. **AGENTS.md Compliance** – Are all layer contracts respected?
2. **Security** – Is auth logic correct? Rate limiting appropriate? OWASP check?
3. **Migrations** – Are new migrations properly tested?
4. **Error Handling** – Do all error responses follow the canonical contract?
5. **Code Quality** – No empty catch blocks, no silent errors, proper layering

Use the following skills to validate:
- `/skill:agendas-compliance` for contract validation
- `/skill:security-audit` for security review
- `/skill:migration-testing` for migration verification
- `/skill:error-contract` for error response validation

${@:-correctness, security, and error handling}