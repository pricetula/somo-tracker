# ADR-0017: Data Protection & Child Privacy

## Metadata

| Field | Value |
|-------|-------|
| id | ADR-0017 |
| title | Data Protection & Child Privacy |
| status | proposed |
| date | 2026-11-25 |
| author | backend, product |
| deciders | backend, product |
| relates_to | SPEC-002, ADR-0013, ADR-0001, ADR-0002 |

## Status

proposed

## Context

Somotracker stores personal data about minors (students): names, dates of birth, attendance, assessment results, guardian contact details. This triggers data protection obligations across East Africa — notably Kenya's Data Protection Act 2019, and analogous laws in Uganda, Tanzania, and Rwanda.

Key obligations: lawful basis for processing, parental consent for minors, data minimization, retention limits, and security of student data.

## Decision

Adopt a **consent-first, minimal-data, retention-limited model**:

1. **Parental/guardian consent**: Before a student's personal data is processed, a guardian (ADR-0013) must provide consent. Consent is recorded as a `consent_records` entry:
   - `id` UUID PK
   - `student_id` UUID → `students.id`
   - `guardian_id` UUID → `guardians.id`
   - `consent_type` TEXT (e.g. `data_processing`, `photo_use`, `report_card_delivery`)
   - `granted` BOOL
   - `granted_at` TIMESTAMPTZ
   - `withdrawn_at` TIMESTAMPTZ (nullable)

2. **Data minimization**: Collect only what is needed for school administration. `students.date_of_birth` is informational only (ADR-0011) and never used for automated placement. No collection of sensitive data (health, ethnicity, religion) in MVP.

3. **Retention policy**: Student records are retained while the student is enrolled plus a defined period (to be configured per country). A future migration will add `deleted_at` soft-delete and archival logic. No indefinite retention.

4. **Multi-tenant isolation**: Student and guardian data is scoped to a school/organization (ADR-0002). No cross-school data sharing.

5. **Guardian data rights**: Guardians can request correction or deletion of their and their child's data via school admin (no self-service portal in MVP).

6. **Security**: Session tokens hashed (ADR-0003), OTP codes HMAC'd (ADR-0001), database access restricted to application tier. No encryption-at-rest requirement beyond platform defaults in MVP.

## Alternatives Considered

| Alternative | Pros | Cons | Why rejected |
|-------------|------|------|--------------|
| Self-service privacy portal | Guardian autonomy | High frontend complexity, not MVP | Rejected — school admin handles requests |
| Full anonymization for analytics | Privacy-preserving | Reduces report card and transcript utility | Rejected — admin use case needs identifiable data |
| Consent-first (chosen) | Lawful basis for minors' data | Adds consent collection step at enrollment | Accepted |

## Consequences

### Positive

- Lawful basis for processing minors' data (guardian consent).
- Data minimization reduces risk exposure.
- Multi-tenant isolation prevents cross-school data leakage.

### Negative

- Consent collection adds friction at student enrollment.
- No self-service data rights portal (guardians must go through school).

### Neutral / Risks

- Retention period not yet defined per country — must be configured before launch in each country.
- Consent withdrawal handling (stop processing, delete or anonymize) not yet implemented in code.

## Links

- Spec: `docs/specs/002-education-system-configuration.md`
- Guardians: `docs/adr/0013-guardian-management.md`
- Multi-tenancy: `docs/adr/0002-multi-tenancy-model.md`
- Session management: `docs/adr/0003-session-management.md`
