# ADR-0018: Fees & Payments (Deferred)

## Metadata

| Field | Value |
|-------|-------|
| id | ADR-0018 |
| title | Fees & Payments |
| status | proposed |
| date | 2026-11-25 |
| author | product |
| deciders | product, backend |
| relates_to | SPEC-002, ADR-0013, ADR-0007 |

## Status

proposed — **deferred, not MVP**

## Context

Private schools in East Africa rely on fee collection, and public schools may have levies. Fee tracking (invoices, payments, balances, bursaries) is a common school administration need. However, it introduces financial data sensitivity, payment integrations (mobile money: M-Pesa, Airtel Money, MTN), and reconciliation complexity.

This ADR is a placeholder acknowledging the domain. Fees and payments are explicitly **out of scope for MVP**.

## Decision

**Defer fees and payments to a post-MVP release.** No fee tables, payment provider integrations, or invoice models will be implemented in the initial release.

When revisited, the following should be considered:
- `fee_structures` (per grade/stream, per term)
- `invoices` (per student, per term)
- `payments` (mobile money integration, manual entry)
- `bursaries` / `scholarships`
- Multi-currency (KES, UGX, TZS, RWF)

## Alternatives Considered

| Alternative | Pros | Cons | Why rejected |
|-------------|------|------|--------------|
| MVP fee tracking | Useful for schools | Financial complexity, payment integration, not core to assessment tracking | Rejected — defer |
| Mobile money integration | Familiar payment method in East Africa | Regulatory (mobile money licenses), reconciliation, fraud risk | Rejected — post-MVP |

## Consequences

### Positive

- Keeps MVP focused on assessment tracking and core configuration.
- Avoids financial data sensitivity and regulatory complexity in initial release.

### Negative

- Schools needing fee tracking must use external tools for MVP.

### Neutral / Risks

- Future fee module will need to integrate with students (ADR-0002/SPEC-002), guardians (ADR-0013), and academic terms (SPEC-002).

## Links

- Spec: `docs/specs/002-education-system-configuration.md`
- Guardians: `docs/adr/0013-guardian-management.md`
