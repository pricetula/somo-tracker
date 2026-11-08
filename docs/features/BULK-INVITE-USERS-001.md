---
id: BULK-INVITE-USERS-001
title: Bulk User Invitations via Stytch Magic Links
status: draft
version: 1.0.0
scope: v1
depends_on: []
touches: [backend, frontend]
---

# Feature

## Goal
Enable school administrators to bulk invite multiple users (admins, teachers, parents) to a school via Stytch magic link invitations. Invitees click a link to complete onboarding, creating their user account and school membership automatically.

## Out of scope
- Student enrollment (students are invited but do not login until later)
- Per-invitee custom role selection (all invitees receive same default role)
- SMS invitations (email only in v1)
- SMS reminders for pending invites
- Advanced invitation customization (branding, templates)
- Invite expiration beyond 7 days (configurable but default 7 days)

## Domain references
- docs/domain/glossary.md (User, School, Membership, Role, Tenant)
- docs/domain/model.md (User, School, Membership)
- docs/features/FEATURE-STYTCH-B2B-ONBOARDING-SESSION-001.md (Stytch magic link flow)

## Acceptance criteria

### AC-1: Invite creation
- Admin can select multiple users (via email list or CSV upload) to invite to a specific school
- Admin must specify a role for each invitee (admin, teacher, parent) - role is mandatory and validated
- System creates a Stytch magic link invitation for each user using the `MagicLinks.Email.Invite()` endpoint with organization_id and email
- Invitees receive an email with a Stytch magic link containing a unique token
- Invite record is created in the database with status "pending" and 7-day expiration
- Role is stored in the invite record and will be used to create the appropriate membership on acceptance

### AC-2: Invite table
- A table shows all invited users for a school with columns:
  - Email
  - Role (admin/teacher/parent)
  - Status (pending, accepted, expired, revoked)
  - Invite date
  - Expiration date
  - Actions: Resend link, Copy link, Delete invite
- Invitees with "accepted" status appear in the school's membership list

### AC-3: Invite acceptance
- When an invitee clicks the magic link, Stytch processes the invitation and authenticates them
- Stytch member is created in the organization with the invited email
- Backend receives Stytch webhook for invitation acceptance OR user completes login flow
- Backend retrieves the invite record by email and organization_id via Stytch lookup
- Backend creates the user account with the Stytch member_id linked
- Backend creates school membership with the **role stored in the invite record** (admin/teacher/parent)
- User appears in the school's membership list with the assigned role from the invitation
- If the user already exists in the system, the new school membership is created with the invited role (no duplicate user)

### AC-4: Resend capability
- Admin can resend a new magic link to any "pending" invitee
- Resending calls Stytch `MagicLinks.Email.Invite()` again with the same organization_id and email
- Stytch creates a new invitation token and sends email
- Resend count is tracked and limited to 3 attempts per invite
- Resent invites maintain the original expiration date (7 days from original creation)
- Resend preserves the original invite record's role assignment

### AC-5: Validation and constraints
- Invitees must belong to the same tenant as the school
- Cannot invite users who already have an active membership in the target school
- Cannot invite users who already have an account in the system (unless they are deleted first)
- Email format validation required
- School must exist and belong to the tenant
- Rate limiting: max 10 invites per hour per admin

### AC-6: Security and audit
- All invitations are logged in the audit log with actor, timestamp, email, and action
- Stytch magic link tokens are never logged or stored in plain text
- Resend actions are audited

## API changes

### New endpoints:

1. **POST /api/v1/schools/{school_id}/invite**
   - Request body: array of invitee objects (email, role)
   - Response: list of invitation IDs and status
   - Creates Stytch magic link invitations and database records

2. **GET /api/v1/schools/{school_id}/invites**
   - Returns list of pending, accepted, and expired invites for the school
   - Pagination support

3. **POST /api/v1/schools/{school_id}/invite/{invite_id}/resend**
   - Triggers Stytch to send a new magic link
   - Updates invite status and resend count

4. **POST /api/v1/auth/invite/accept** (Stytch webhook handler)
   - Receives Stytch invitation acceptance event
   - Creates user and membership on the school
   - Returns 200 OK on success

### API contract changes:
- Add `invite` field to user creation response (optional)
- Add `invite_status` field to user profile endpoints
- Update `User` DTO to include `invite_status` and `invite_id` fields (optional)

## Backend notes

### New database tables:
- `invite_records`:
  - `id` (uuidv7 PK)
  - `school_id` (FK)
  - `tenant_id` (uuid, FK for RLS)
  - `email` (varchar, normalized)
  - `role` (enum: admin, teacher, parent) - MANDATORY, used for membership creation
  - `status` (enum: pending, accepted, expired, revoked)
  - `created_at` (timestamptz)
  - `expires_at` (timestamptz, 7 days from creation)
  - `stytch_organization_id` (uuid, Stytch organization ID)
  - `stytch_invitation_id` (uuid, Stytch invitation ID)
  - `stytch_member_id` (uuid, Stytch member ID, populated after acceptance)
  - `resend_count` (integer, default 0)
  - `last_resent_at` (timestamptz, nullable)
  - `created_by` (uuid, FK to user, audit trail)
  - `updated_at` (timestamptz)

### Stytch integration (v18):
- Use `client.MagicLinks.Email.Invite()` endpoint to create invitations
- **Parameters:** `organization_id` (from school's tenant organization), `email`, `user_role` (optional, but we store our own role mapping)
- Store the Stytch organization_id in invite record for correlation
- When invitee accepts: Stytch authenticates them and creates/updates member in organization
- Backend webhook or login flow must lookup invite record by email + organization_id
- **Critical:** Role is retrieved from the invite record, NOT from Stytch (Stytch doesn't store our custom roles)
- Use existing `CreateUser` and `CreateMembership` functions from identity service
- Ensure tenant isolation: only invite users to schools in the same tenant

### Role mapping flow:
1. Admin creates invite with role (admin/teacher/parent) - **MANDATORY**
2. Role is stored in `invite_records.role` column
3. Stytch invitation is sent (Stytch has no role concept for invitations)
4. User accepts invitation → Stytch member created
5. Backend looks up invite by email + organization_id → retrieves stored role
6. Backend creates school membership using the stored role
7. Role is never lost or ambiguous

### Validation:
- Validate school existence and tenant membership
- Validate email format and uniqueness (skip if user already exists in system)
- Ensure invitee role is appropriate for the school (admin/teacher/parent)
- Rate limiting: 10 invites/hour per admin user

### Error handling:
- Return 400 for invalid emails or duplicate invites
- Return 403 if admin lacks permission to invite users to the school
- Return 429 if rate limit exceeded
- Return 409 if invite already exists or user already has active membership

## Frontend notes

### New UI components:
- **Invite Users Page**: 
  - School selector dropdown
  - Bulk invite form (email list or CSV upload)
  - Role selection per invitee (optional)
  - Submit button
- **Invite Table**:
  - Lists all invites for the school
  - Status indicators (color-coded)
  - Action buttons: Resend, Copy Link, Delete
- **Resend Flow**:
  - Confirmation modal before resending
  - Success notification on resend completion

### Integration points:
- Stytch magic link flow (existing onboarding flow) used for invitees
- Stytch webhook integration for invitation acceptance events
- Stytch SDK already available in backend (`platform/stytch`)

### UX considerations:
- Clear indication of invite expiration (e.g., "Expires in 5 days")
- Resend button disabled after 3 attempts
- Email template shows school name and logo
- Success message after invite sent: "Invitation sent to X users. They will receive an email shortly."

## Testing & Verification

### Unit Tests (AC → Test mapping)

**AC-1: Invite Creation**
- **Test_AC1_01_ValidInviteCreation**: Create invite with valid email and role → succeeds
- **Test_AC1_02_InvalidEmailFormat**: Create invite with invalid email → returns 400 with error code `INVALID_EMAIL_FORMAT`
- **Test_AC1_03_RoleMandatory**: Create invite without role → returns 400 with error code `ROLE_REQUIRED`
- **Test_AC1_04_DuplicateInvite**: Create invite for same email/school → returns 409 with error code `INVITE_ALREADY_EXISTS`
- **Test_AC1_05_RateLimitExceeded**: Create 11 invites in one hour → returns 429 with error code `RATE_LIMIT_EXCEEDED`
- **Test_AC1_06_EmailNormalization**: Create invite with `John@Example.COM` → stores as `john@example.com`
- **Test_AC1_07_StytchSDKCall**: Verify `client.MagicLinks.Email.Invite()` called with correct org_id, email
- **Test_AC1_08_TenantIsolation**: Try to invite user to school in different tenant → returns 404

**AC-2: Invite Table**
- **Test_AC2_01_InviteListing**: GET invites returns pending, accepted, expired with correct fields
- **Test_AC2_02_InvitePagination**: GET invites with pagination returns correct items
- **Test_AC2_03_InviteStatusFilters**: Filter invites by status (pending/accepted/expired)
- **Test_AC2_04_InviteFieldsComplete**: Verify invite includes email, role, status, created_at, expires_at
- **Test_AC2_05_AcceptedInvitesShowMembership**: Accepted invitee appears in school membership list

**AC-3: Invite Acceptance**
- **Test_AC3_01_WebhookProcessing**: Stytch webhook received → user created with correct role from invite record
- **Test_AC3_02_UserAlreadyExists**: Existing user accepts invite → new membership created, no duplicate user
- **Test_AC3_03_RoleFromInviteRecord**: Verify membership role matches `invite_records.role`, not hardcoded
- **Test_AC3_04_InviteRecordUpdated**: After acceptance, invite status changes to "accepted" and stytch_member_id populated
- **Test_AC3_05_MultipleSchoolsSameUser**: User accepts invites to multiple schools → multiple memberships created
- **Test_AC3_06_ExpiredInviteRejected**: User accepts expired invite → returns error, no user created

**AC-4: Resend Capability**
- **Test_AC4_01_ResendPendingInvite**: Resend pending invite → Stytch called, resend_count incremented
- **Test_AC4_02_ResendLimitEnforced**: Resend 3 times → 4th resend returns 429 with error code `RESEND_LIMIT_EXCEEDED`
- **Test_AC4_03_ResendExpiredInvite**: Try to resend expired invite → returns 400 with error code `INVITE_EXPIRED`
- **Test_AC4_04_ResendPreservesRole**: Resend invite → original role preserved
- **Test_AC4_05_ResendAuditLog**: Resend action logged in audit_log with actor, email, timestamp

**AC-5: Validation & Constraints**
- **Test_AC5_01_InvalidRoleRejected**: Invite with role "invalid" → returns 400 with error code `INVALID_ROLE`
- **Test_AC5_02_AdminPermissionCheck**: Non-admin tries to invite → returns 403 with error code `FORBIDDEN`
- **Test_AC5_03_ExistingMembershipBlocked**: Invite user with existing membership → returns 409 with error code `USER_ALREADY_MEMBER`
- **Test_AC5_04_SchoolExistsValidation**: Invite to non-existent school → returns 404 with error code `SCHOOL_NOT_FOUND`
- **Test_AC5_05_CrossTenantBlocked**: Invite user to school in different tenant → returns 404

**AC-6: Security & Audit**
- **Test_AC6_01_AuditLogCreated**: Invite creation logged with actor, email, action, timestamp
- **Test_AC6_02_TokensNotLogged**: Verify Stytch tokens never appear in logs or error messages
- **Test_AC6_03_RLSEnforced**: Query invites returns only invites for current tenant
- **Test_AC6_04_TokenHashing**: Verify Stytch tokens are hashed or not stored in plain text

### Integration Tests

**Test_Integration_StytchFlow**: End-to-end invitation flow
1. Create invite via API → verify DB record created
2. Mock Stytch webhook → verify user and membership created with correct role
3. Verify email sent via Stytch mock
4. Cleanup

**Test_Integration_MultiInvite**: Create 10 invites in bulk
- Verify all 10 invites created
- Verify Stytch called 10 times
- Verify rate limiting not triggered

**Test_Integration_ResendFlow**: Complete resend workflow
- Create invite
- Resend 3 times
- 4th resend returns error
- Verify DB state

**Test_Integration_TenantIsolation**: Cross-tenant access
- User from Tenant A tries to access invites for Tenant B
- Returns 404 (not found)

### User Acceptance Scenarios

**UAS-1: First-time bulk invite**
1. Admin logs in to school dashboard
2. Navigates to "Invite Users" page
3. Selects 3 users: `teacher1@example.com` (teacher), `parent1@example.com` (parent), `admin2@example.com` (admin)
4. Clicks "Send Invitations"
5. Sees success message: "Invitations sent to 3 users"
6. Invite table shows 3 rows with status "pending" and role listed
7. Invitees receive emails with magic links
8. Each invitee clicks link → authenticates with Stytch → appears in school membership with correct role

**UAS-2: Resend invitation**
1. Admin views invite table
2. Sees `teacher1@example.com` with status "pending" and 5 days until expiration
3. Clicks "Resend" button
4. Confirmation modal appears
5. Admin confirms → resend count increments to 1
6. `teacher1@example.com` receives new magic link email
7. Admin can resend up to 2 more times

**UAS-3: Prevent duplicate invites**
1. Admin tries to invite `teacher1@example.com` again for same school
2. Sees error: "Invite already exists for this email"
3. Admin can view existing invite instead

**UAS-4: Role mapping verification**
1. Admin creates invite for `newteacher@example.com` with role "teacher"
2. User accepts invitation
3. User appears in school membership list with role "teacher"
4. User cannot access admin-only features
5. Admin can change role in school management if needed

**UAS-5: Cross-tenant protection**
1. Admin from School A tries to invite user to School B (different tenant)
2. Sees error: "School not found" (security through obscurity)
3. No cross-tenant data leakage

### Compliance Checklist

**Code Quality**
- [ ] `make compliance` passes
- [ ] `go vet ./...` passes
- [ ] `golangci-lint run ./...` passes
- [ ] All new endpoints have OpenAPI spec updates in `docs/api/openapi.yaml`
- [ ] `make generate` is idempotent (no diff after running)

**Security**
- [ ] Error contract respected (ARCH-ERROR-HANDLING)
- [ ] Tenant RLS enforced on all invite queries
- [ ] Stytch tokens never logged
- [ ] Rate limiting enforced (10 invites/hour)
- [ ] Resend limit enforced (3 attempts)
- [ ] RBAC enforced (only admins can invite)

**Testing**
- [ ] Unit tests cover all ACs with AC IDs in test names
- [ ] Integration tests verify Stytch SDK integration
- [ ] Tenant isolation tests pass
- [ ] User acceptance scenarios verified manually
- [ ] Test coverage >80% for new code

**Documentation**
- [ ] OpenAPI spec updated with new endpoints
- [ ] Frontend API types generated
- [ ] Feature spec status updated to `approved` (after human review)
- [ ] README updated with new feature

### Manual Test Procedure

**Pre-requisites**
- Stytch test environment configured
- Test school created with admin user
- Test emails configured

**Steps**
1. Admin creates bulk invite for 3 users (admin, teacher, parent) → verify emails sent (mock)
2. Simulate invitee clicking link (via Stytch test mode) → verify user created with correct role
3. Verify user appears in school membership list with correct role
4. Admin resends invite to one user → verify new magic link sent
5. Verify invite table shows correct status and resend count
6. Test rate limiting (11 invites in 1 hour)
7. Test tenant isolation (cross-tenant access)
8. Test invite expiration (7 days)
9. Verify audit logs created for all actions

### Definition of Done

- All acceptance criteria covered by tests citing AC IDs
- docs/api/openapi.yaml updated with new endpoints
- Generated API types current and idempotent
- Error contract respected (ARCH-ERROR-HANDLING)
- Docs updated in same change
- `make compliance` passes
- Status updated: implemented by agent, verified by human

## Open questions

- Role assignment: Should invitees receive a default role (e.g., admin) or allow selection per invitee?
- Invite expiration: Is 7 days sufficient, or should it be configurable per school?
- What happens if a user accepts an invite but already has an account in another school?
- Should we support multiple schools per user (e.g., parent with children in multiple schools)?
- How to handle Stytch rate limits during bulk invitation creation?
- Should we store the Stytch invitation ID for audit purposes?

---