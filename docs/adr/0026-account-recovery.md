# ADR-0026: Account Recovery

## Metadata

| Field | Value |
|-------|-------|
| id | ADR-0026 |
| title | Account Recovery |
| status | proposed |
| date | 2026-11-25 |
| author | backend, product |
| deciders | backend, product |
| relates_to | ADR-0001, ADR-0002, ADR-0017 |

## Status

proposed

## Context

Somotracker uses passwordless OTP authentication (ADR-0001). Users authenticate exclusively via email OTP — there is no password to reset. If a user loses access to their email account (compromised, deleted, or forgotten), they cannot authenticate and there is no password reset flow.

There is no documented account recovery path. Without one, a user who loses email access is permanently locked out of their account and all associated school data.

## Decision

Adopt a **guardian-assisted, admin-mediated account recovery model**:

1. **No self-service recovery**: Users cannot recover their own account without email access. No security questions, no backup codes, no alternative email.

2. **Guardian-assisted recovery**: If a guardian (ADR-0013) is linked to the student/user, the guardian can contact the school administrator to initiate recovery. The school admin verifies the guardian's identity (phone, in-person, government ID) and requests account recovery from Somotracker support.

3. **Support-mediated recovery**: Somotracker support verifies the school admin's identity, then:
   - Verifies the guardian's relationship to the student via `student_guardians` table.
   - Updates the user's `normalized_email` to a new email address provided by the guardian/school.
   - Invalidates all existing sessions for the user (sets `revoked_at` on all `sessions` rows).
   - The user can then authenticate with the new email via OTP.

4. **Data protection compliance**: Recovery requires guardian consent (ADR-0017 `consent_records`). The recovery event is logged (who, when, what changed) without logging the new email value.

5. **Organization owner recovery**: If the organization owner loses email access, another user with `role = 'admin'` in the same organization can initiate recovery (ADR-0002, ADR-0019 RBAC). If no admin exists, the organization is locked.

## Alternatives Considered

| Alternative | Pros | Cons | Why rejected |
|-------------|------|------|--------------|
| Self-service recovery (security questions) | User autonomy | Security questions are weak, not appropriate for minors' data | Rejected — security and child privacy concerns |
| Backup codes | Works without email | Users lose backup codes; adds UX friction | Rejected — defer to future |
| Guardian-assisted, admin-mediated (chosen) | Fits school administration model, verified identity | Requires human intervention, slower than self-service | Accepted — matches East African school administration patterns |
| No recovery (account is lost forever) | Simplest, most secure | Users permanently locked out on email loss | Rejected — unacceptable user experience |

## Consequences

### Positive

- Verified identity before email change (prevents social engineering).
- Fits school administration model (guardian + school admin verification).
- Compliant with child data protection (guardian consent required).

### Negative

- Recovery requires human intervention (slower than self-service).
- Depends on school admin availability.

### Neutral / Risks

- Social engineering risk if school admin is compromised — mitigated by multi-person verification.
- Recovery audit logging must not log PII (new email address).

## Implementation Notes

1. **Recovery endpoint**: `POST /api/v1/auth/recovery` (support-only, authenticated, requires admin role).
2. **Email change**: Update `users.normalized_email` + `users.email`, invalidate all sessions.
3. **Audit log**: Record recovery events in an immutable log (school ID, guardian ID, admin ID, timestamp, action type) — without new email value.
4. **Rate limit**: Recovery endpoint strictly rate-limited (ADR-0005) to prevent abuse.

## Links

- Authentication: `docs/adr/0001-authentication-architecture.md`
- Multi-tenancy: `docs/adr/0002-multi-tenancy-model.md`
- Guardian management: `docs/adr/0013-guardian-management.md`
- Data protection: `docs/adr/0017-data-protection-child-privacy.md`
- RBAC: `docs/adr/0019-rbac-authorization.md`
