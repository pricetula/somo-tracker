# Model Glossary

Canonical domain model and terminology for the Somotracker auth revamp.

Source of truth for entity names, attribute names, and relationships. All spec
docs must use these names; do not invent aliases.

## Status

proposed

## Entities

### organization
Top-level customer account and data-isolation boundary. Replaces the old
"tenant" concept.

| Attribute | Type | Null | Notes |
|-----------|------|------|-------|
| id | UUID | NOT NULL | Primary key, uuidv7 |
| name | TEXT | NOT NULL | Organization display name |
| status | TEXT | NOT NULL | `active`, `suspended`, `revoked` |
| created_at | TIMESTAMPTZ | NOT NULL | Default now() |
| updated_at | TIMESTAMPTZ | NOT NULL | Default now() |
| deleted_at | TIMESTAMPTZ | NULL | Soft delete |

### user
One global identity per normalized email. A user belongs to exactly one
organization.

| Attribute | Type | Null | Notes |
|-----------|------|------|-------|
| id | UUID | NOT NULL | Primary key, uuidv7 |
| organization_id | UUID | NOT NULL | FK → organizations.id |
| email | TEXT | NOT NULL | Raw email as provided |
| normalized_email | TEXT | NOT NULL | Lowercased, trimmed. UNIQUE |
| status | TEXT | NOT NULL | `active`, `suspended`, `revoked` |
| email_verified_at | TIMESTAMPTZ | NULL | Set when OTP verified |
| created_at | TIMESTAMPTZ | NOT NULL | Default now() |
| updated_at | TIMESTAMPTZ | NOT NULL | Default now() |

### school
A school owned by exactly one organization.

| Attribute | Type | Null | Notes |
|-----------|------|------|-------|
| id | UUID | NOT NULL | Primary key, uuidv7 |
| organization_id | UUID | NOT NULL | FK → organizations.id |
| name | TEXT | NOT NULL | School display name |
| status | TEXT | NOT NULL | `active`, `suspended`, `revoked` |
| created_at | TIMESTAMPTZ | NOT NULL | Default now() |
| updated_at | TIMESTAMPTZ | NOT NULL | Default now() |

### user_school
Links a user to a school within their organization. A user may have
different roles in different schools.

| Attribute | Type | Null | Notes |
|-----------|------|------|-------|
| id | UUID | NOT NULL | Primary key, uuidv7 |
| organization_id | UUID | NOT NULL | FK → organizations.id |
| user_id | UUID | NOT NULL | FK → users.id |
| school_id | UUID | NOT NULL | FK → schools.id |
| role | TEXT | NOT NULL | School-level role (e.g. `OWNER`, `ADMIN`, `TEACHER`) |
| status | TEXT | NOT NULL | `active`, `revoked` |
| created_at | TIMESTAMPTZ | NOT NULL | Default now() |
| revoked_at | TIMESTAMPTZ | NULL | Set when assignment revoked |

### session
Opaque server-side session. Raw token never stored; only SHA-256 hash.

| Attribute | Type | Null | Notes |
|-----------|------|------|-------|
| id | UUID | NOT NULL | Primary key, uuidv7 |
| token_hash | BYTEA | NOT NULL | SHA-256 of raw token, UNIQUE |
| user_id | UUID | NOT NULL | FK → users.id |
| organization_id | UUID | NOT NULL | Denormalized for efficient scoping |
| active_school_id | UUID | NULL | FK → schools.id. NULL for org-level access |
| created_at | TIMESTAMPTZ | NOT NULL | Default now() |
| last_seen_at | TIMESTAMPTZ | NULL | Sliding idle window |
| expires_at | TIMESTAMPTZ | NOT NULL | Absolute expiry |
| revoked_at | TIMESTAMPTZ | NULL | Set on logout/revocation |
| revocation_reason | TEXT | NULL | Why revoked |

### login_challenge
Short-lived email OTP challenge. Code never stored in plaintext.

| Attribute | Type | Null | Notes |
|-----------|------|------|-------|
| id | UUID | NOT NULL | Primary key, uuidv7 |
| email_normalized | TEXT | NOT NULL | Normalized email |
| purpose | TEXT | NOT NULL | `login` |
| code_mac | BYTEA | NOT NULL | HMAC-SHA256(server_secret, challenge_id \|\| code) |
| expires_at | TIMESTAMPTZ | NOT NULL | 10 minutes from creation |
| attempt_count | INTEGER | NOT NULL | Default 0 |
| max_attempts | INTEGER | NOT NULL | Default 5 |
| used_at | TIMESTAMPTZ | NULL | Set on successful verification |
| invalidated_at | TIMESTAMPTZ | NULL | Set on resend or expiry |
| created_at | TIMESTAMPTZ | NOT NULL | Default now() |

### email_outbox
Transactional outbox for email delivery. Challenge and outbox committed
in the same DB transaction.

| Attribute | Type | Null | Notes |
|-----------|------|------|-------|
| id | UUID | NOT NULL | Primary key, uuidv7 |
| challenge_id | UUID | NOT NULL | FK → login_challenges.id |
| message_type | TEXT | NOT NULL | `otp`, `invitation` |
| status | TEXT | NOT NULL | `pending`, `sent`, `failed` |
| attempts | INTEGER | NOT NULL | Default 0 |
| next_attempt_at | TIMESTAMPTZ | NULL | For retry scheduling |
| provider_message_id | TEXT | NULL | Brevo message ID |
| encrypted_payload | BYTEA | NULL | Encrypted OTP + email, key outside DB |
| created_at | TIMESTAMPTZ | NOT NULL | Default now() |
| sent_at | TIMESTAMPTZ | NULL | Set on successful delivery |

### guardian
A parent or legal caregiver linked to one or more students. Guardians are
contacts, not system users (no login, no OTP in MVP).

| Attribute | Type | Null | Notes |
|-----------|------|------|-------|
| id | UUID | NOT NULL | Primary key, uuidv7 |
| school_id | UUID | NOT NULL | FK → schools.id |
| full_name | TEXT | NOT NULL | Guardian display name |
| phone | TEXT | NOT NULL | Primary contact number |
| email | TEXT | NULL | Optional email address |
| relationship | TEXT | NOT NULL | e.g. `father`, `mother`, `guardian`, `aunt`, `uncle` |
| address | TEXT | NULL | Mailing address |
| is_emergency_contact | BOOL | NOT NULL | Default false |
| created_at | TIMESTAMPTZ | NOT NULL | Default now() |
| updated_at | TIMESTAMPTZ | NOT NULL | Default now() |

### student_guardian
Links a student to a guardian. A student may have multiple guardians.

| Attribute | Type | Null | Notes |
|-----------|------|------|-------|
| id | UUID | NOT NULL | Primary key, uuidv7 |
| student_id | UUID | NOT NULL | FK → students.id |
| guardian_id | UUID | NOT NULL | FK → guardians.id |
| is_primary | BOOL | NOT NULL | Default false |
| created_at | TIMESTAMPTZ | NOT NULL | Default now() |

### attendance_record
One row per student per school day. Records daily attendance status.

| Attribute | Type | Null | Notes |
|-----------|------|------|-------|
| id | UUID | NOT NULL | Primary key, uuidv7 |
| student_id | UUID | NOT NULL | FK → students.id |
| stream_id | UUID | NOT NULL | FK → streams.id |
| academic_year_id | UUID | NOT NULL | FK → school_academic_years.id |
| attendance_date | DATE | NOT NULL | Date of attendance |
| status | TEXT | NOT NULL | `present`, `absent`, `late`, `excused` |
| recorded_by_teacher_id | UUID | NOT NULL | FK → teachers.id |
| recorded_at | TIMESTAMPTZ | NOT NULL | Default now() |

### teacher_qualification
Records a teacher's qualification to teach a subject at a level.

| Attribute | Type | Null | Notes |
|-----------|------|------|-------|
| id | UUID | NOT NULL | Primary key, uuidv7 |
| teacher_id | UUID | NOT NULL | FK → teachers.id |
| subject_code | TEXT | NOT NULL | Subject code (e.g. `MATH`, `ENG`) |
| education_level_id | UUID | NOT NULL | FK → education_levels.id |
| qualification | TEXT | NOT NULL | e.g. `B.Ed Mathematics` |
| registration_number | TEXT | NULL | Teacher council registration number |
| created_at | TIMESTAMPTZ | NOT NULL | Default now() |

### consent_record
Records guardian consent for processing a student's personal data.

| Attribute | Type | Null | Notes |
|-----------|------|------|-------|
| id | UUID | NOT NULL | Primary key, uuidv7 |
| student_id | UUID | NOT NULL | FK → students.id |
| guardian_id | UUID | NOT NULL | FK → guardians.id |
| consent_type | TEXT | NOT NULL | e.g. `data_processing`, `photo_use`, `report_card_delivery` |
| granted | BOOL | NOT NULL | Whether consent was granted |
| granted_at | TIMESTAMPTZ | NOT NULL | When consent was granted |
| withdrawn_at | TIMESTAMPTZ | NULL | When consent was withdrawn |

## Relationships

- `users.organization_id → organizations.id` (CASCADE)
- `schools.organization_id → organizations.id` (CASCADE)
- `user_schools.organization_id, user_schools.user_id → users(organization_id, id)` (composite FK)
- `user_schools.organization_id, user_schools.school_id → schools(organization_id, id)` (composite FK)
- `sessions.user_id → users.id` (CASCADE)
- `sessions.organization_id → organizations.id` (CASCADE)
- `sessions.active_school_id → schools.id` (NULL allowed)
- `login_challenges.email_normalized` matches `users.normalized_email` (no FK; challenge may pre-exist user)
- `email_outbox.challenge_id → login_challenges.id` (CASCADE)
- `students.school_id → schools.id` (CASCADE)
- `streams.school_id → schools.id` (CASCADE)
- `streams.grade_year_id → school_grade_years.id` (CASCADE)
- `streams.academic_year_id → school_academic_years.id` (CASCADE)
- `teachers.school_id → schools.id` (CASCADE)
- `guardians.school_id → schools.id` (CASCADE)
- `student_guardians.student_id → students.id` (CASCADE)
- `student_guardians.guardian_id → guardians.id` (CASCADE)
- `attendance_records.student_id → students.id` (CASCADE)
- `attendance_records.stream_id → streams.id` (CASCADE)
- `attendance_records.academic_year_id → school_academic_years.id` (CASCADE)
- `attendance_records.recorded_by_teacher_id → teachers.id` (CASCADE)
- `teacher_qualifications.teacher_id → teachers.id` (CASCADE)
- `teacher_qualifications.education_level_id → education_levels.id` (CASCADE)
- `consent_records.student_id → students.id` (CASCADE)
- `consent_records.guardian_id → guardians.id` (CASCADE)

## Terminology

| Term | Meaning | Notes |
|------|---------|-------|
| `organization_id` | Top-level account ID | Never `tenant_id` |
| opaque token | Session random string | Set in `__Host-session` cookie by Go backend |
| `token_hash` | SHA-256 of opaque token | Stored in DB, never raw token |
| OTP | Six-digit code | HMAC stored, never plaintext |
| code_mac | HMAC-SHA256(server_secret, challenge_id \|\| code) | Key from Doppler |
| Turnstile | Cloudflare bot protection | Disable via env var for local dev |
| Brevo | Transactional email provider | API key from Doppler |
| BFF | Next.js App Router | Proxies to Go, forwards cookies |
| proxy | Next.js → Go | Browser never calls Go directly |
| guardian | Parent or legal caregiver linked to students | Contact only, no system login in MVP |
| attendance_record | Daily per-student attendance status | present / absent / late / excused |
| consent_record | Guardian consent for processing student data | Can be withdrawn at any time |

## Constraints

- `organizations.name` unique
- `users.normalized_email` unique (global identity)
- `schools.(organization_id, name)` unique
- `user_schools.(user_id, school_id)` unique
- Composite FKs prevent cross-organization user↔school assignment
- `login_challenges.expires_at` 10 minutes after `created_at`
- `sessions.expires_at` 30 days after `created_at` (absolute)
- Sessions use absolute TTL only; no sliding idle timeout is enforced in MVP

## Links

- Authentication architecture: `docs/adr/0001-authentication-architecture.md`
- Session management: `docs/adr/0003-session-management.md`
- Data protection & child privacy: `docs/adr/0017-data-protection-child-privacy.md`
- Guardian management: `docs/adr/0013-guardian-management.md`
- Attendance tracking: `docs/adr/0012-attendance-tracking.md`
- Specification lifecycle: proposed → approved → implemented → done
