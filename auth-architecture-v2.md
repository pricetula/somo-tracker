# Authentication & Authorization Architecture

## Version 2 --- Organization and Multi-School Model

**Status:** Architecture specification draft\
**Scope:** Email OTP authentication, server-side sessions, organization
isolation, school-level access, invitations, and authorization\
**Primary backend:** Go\
**Frontend/BFF:** Next.js\
**Database:** PostgreSQL\
**Edge services:** Cloudflare\
**Timezone:** Africa/Nairobi for application-level display and
scheduling where relevant

------------------------------------------------------------------------

## 1. Purpose

This document defines the authentication and authorization architecture
for the school management platform.

The design uses **organization** as the top-level account and
data-isolation boundary. It does not use the term *tenant* in
product-facing or technical domain terminology.

The system supports organizations that operate multiple schools. A user
belongs to exactly one organization but may be granted access to
multiple schools within that organization.

The design separates:

1.  **Authentication:** proving which user is signing in.
2.  **Organization membership:** establishing the one organization to
    which that user belongs.
3.  **School access:** determining which schools within that
    organization the user may access.
4.  **Authorization:** determining which actions the user may perform at
    organization and school level.
5.  **Session management:** maintaining and revoking authenticated
    browser sessions.

The Go backend is authoritative for all five areas.

## 2. Core architectural decisions

These are foundational invariants. Implementation and future changes
must preserve them.

### 2.1 Organization and user relationships

-   An organization may contain many users.
-   An organization may contain many schools.
-   Every user belongs to exactly one organization.
-   A user cannot belong to multiple organizations.
-   Every school belongs to exactly one organization.
-   A user may have access to zero, one, or many schools within their
    organization.
-   A user must never be granted access to a school belonging to another
    organization.
-   Organization ownership and school access are separate concepts.
-   A user's organization-level permissions do not automatically imply
    access to every school unless an explicit organization-wide
    permission grants that access.

### 2.2 Terminology

  -----------------------------------------------------------------------
  Term                                Meaning
  ----------------------------------- -----------------------------------
  Organization                        Top-level customer/account and
                                      data-isolation boundary

  User                                A person with one global identity
                                      and one organization

  School                              A school owned by an organization

  School assignment                   A user's authorization to access a
                                      particular school

  Organization role                   Permissions that apply across the
                                      organization

  School role                         Permissions that apply within a
                                      particular school

  Session                             Server-side authenticated browser
                                      session

  Login challenge                     Short-lived email OTP verification
                                      challenge
  -----------------------------------------------------------------------

Use `organization_id`, not `tenant_id`, in schema, APIs, code, logs, and
documentation.

### 2.3 Identity and email uniqueness

The recommended default is one global user identity per normalized email
address:

-   Store a normalized email value for matching.
-   Enforce a unique database constraint on the normalized email.
-   Do not create a second user identity for the same email in another
    organization.
-   Do not silently move an existing user to another organization when
    an invitation is received.
-   An organization transfer, if ever supported, must be an explicit
    administrative workflow with audit history and session revocation.

If the product later needs shared email addresses across separate people
or organizations, that requires a deliberate identity-model change. It
should not be handled by weakening uniqueness informally.

## 3. System architecture and trust boundaries

### 3.1 Components

  -----------------------------------------------------------------------
  Component               Responsibilities        Must not do
  ----------------------- ----------------------- -----------------------
  Next.js BFF             Render UI, forward API  Make authoritative
                          requests, forward       identity, organization,
                          cookies, handle         school, role, or
                          browser-facing          permission decisions
                          redirects and cache     
                          controls                

  Go API                  OTP, identity, session  Trust organization,
                          lifecycle, organization school, user, or role
                          and school              values supplied by the
                          authorization,          browser
                          invitations, account    
                          lifecycle               

  PostgreSQL              Persist identities,     Rely solely on
                          organizations, schools, application query
                          assignments,            filters for
                          challenges, sessions,   organization isolation
                          invitations, outbox and 
                          audit events; enforce   
                          constraints and RLS     

  Redis                   Distributed rate-limit  Become the source of
                          counters and            truth for users,
                          short-lived operational memberships, or
                          coordination where      sessions
                          appropriate             

  Email worker            Deliver queued          Log OTPs, raw tokens,
                          authentication and      or other secrets
                          invitation emails       

  Cloudflare              DNS, Turnstile, edge    Be assumed to provide
                          controls                
  Brevo                   Email delivery          Be assumed to provide
                          (transactional OTP and   complete application
                          invitation emails)       authorization
  -----------------------------------------------------------------------

### 3.2 Trust rules

-   The browser is untrusted.
-   Next.js is a BFF, not the authorization authority.
-   Go derives the user and organization from the validated server-side
    session.
-   Go validates every requested school against the user's organization
    and school assignments.
-   PostgreSQL RLS provides defense in depth for organization-owned
    data.
-   A client-supplied `organization_id`, `user_id`, `role`, or
    `school_id` is a request, never proof of authorization.
-   The origin must not be reachable in a way that bypasses required
    edge controls unless equivalent protections are enforced at the
    origin.

## 4. Authentication flow: email OTP

### 4.1 Overview

The sign-in process uses a short-lived, single-use email OTP.

1.  User enters an email address in the Next.js UI.
2.  Next.js sends the email and Turnstile token to Go.
3.  Go validates the request and verifies the Turnstile token
    server-side.
4.  Go applies rate limits.
5.  Go creates a login challenge and a durable email-outbox record.
6.  A worker sends the OTP email.
7.  User submits the OTP.
8.  Go verifies the challenge atomically.
9.  Go establishes the authenticated user identity and determines the
    user's organization.
10. Go determines the schools the user may access.
11. Go creates or finalizes a session with an appropriate school
    context.
12. Next.js receives the session cookie and continues to the
    application.

### 4.2 Requesting an OTP

Example endpoint:

`POST /api/v1/auth/otp/request`

Request body:

``` json
{
  "email": "person@example.com",
  "turnstile_token": "..."
}
```

Processing requirements:

1.  Validate request shape and email syntax.
2.  Normalize the email consistently.
3.  Verify Turnstile using the server-side Siteverify API.
4.  Validate the expected hostname and action where configured, as well
    as successful verification and token freshness.
5.  Apply per-email, per-IP, and global rate limits.
6.  Avoid disclosing whether an account exists.
7.  For an eligible account, create a login challenge and email-outbox
    record in one database transaction.
8.  Return a generic response that does not reveal account existence.

A generic response can say that, if the address is eligible, a code will
be sent. Do not return distinguishable messages for registered and
unregistered addresses.

The system may choose not to send an email for unknown addresses, but
the externally visible response and timing should not create an easy
account-enumeration oracle.

### 4.3 OTP generation and storage

Recommended initial policy:

  -----------------------------------------------------------------------
  Property                            Policy
  ----------------------------------- -----------------------------------
  Code                                Six decimal digits

  Generation                          Cryptographically secure random
                                      generator

  Validity                            10 minutes

  Maximum verification attempts       5

  Use                                 Single use

  Resend                              New challenge invalidates prior
                                      active challenge for the same user
                                      and purpose

  Storage                             HMAC of challenge identifier and
                                      code

  Secret                              HMAC key held outside PostgreSQL in
                                      managed secret storage

  Logging                             Never log the code or full
                                      authentication payload
  -----------------------------------------------------------------------

Do not store a plain SHA-256 hash of a six-digit code. The search space
is small enough that a database-only leak could permit offline guessing.

Use a keyed construction such as:

`HMAC-SHA256(server_secret, challenge_id || code)`

Use constant-time comparison when comparing MAC values.

The challenge should include a purpose, for example `login`, so a code
issued for one workflow cannot be reused for another.

### 4.4 Challenge verification

Example endpoint:

`POST /api/v1/auth/otp/verify`

Request body:

``` json
{
  "challenge_id": "uuid",
  "code": "123456"
}
```

The challenge ID is an opaque identifier, not an authorization
credential.

Verification must be atomic. Use a transaction with row locking or an
equivalent conditional update to prevent concurrent requests from:

-   Exceeding the attempt limit.
-   Consuming a challenge more than once.
-   Both succeeding with the same code.
-   Accepting a code after expiry.
-   Accepting a challenge already marked as used or invalidated.

On successful verification:

-   Mark the challenge as used.
-   Record verification time.
-   Resolve the user identity.
-   Resolve the user's single organization.
-   Continue to school-context resolution.

On failure, return a generic error that does not reveal whether the
email, challenge, or code was the specific cause. Internally, record a
safe failure category for monitoring.

### 4.5 Resend behavior

Example endpoint:

`POST /api/v1/auth/otp/resend`

Requirements:

-   Enforce a resend cooldown, initially 60 seconds.
-   Apply per-email and per-IP limits.
-   Invalidate the prior active challenge when a replacement challenge
    is created.
-   Create the replacement challenge and outbox message transactionally.
-   Do not reset abuse counters simply because a new challenge is
    issued.
-   Keep responses generic.
-   Do not allow unlimited resend attempts.

The exact cooldown and limits should be tuned using observed abuse and
delivery metrics.

## 5. Organization and school context

### 5.1 No organization selection at login

A user belongs to exactly one organization. Therefore, the login flow
must not ask the user to choose an organization.

After OTP verification, Go obtains the organization from the
authenticated user's database record.

The client must not choose or override the user's organization.

### 5.2 School selection

A user may have access to multiple schools within their organization.

After authentication:

1.  Go loads the user's active school assignments.
2.  Go confirms each assigned school belongs to the user's organization.
3.  If the user has one accessible school, Go may select it
    automatically.
4.  If the user has multiple accessible schools, the UI presents a
    school selector.
5.  Go validates the selected school again when the selection is
    submitted.
6.  Go stores the active school context in the server-side session.

Example endpoint:

`POST /api/v1/auth/school/select`

Request:

``` json
{
  "school_id": "uuid"
}
```

The endpoint must not accept an organization ID as an authority-bearing
input. The organization comes from the authenticated session.

### 5.3 Users with no school assignment

Do not assume that every authenticated user must have a school
assignment.

Some organization-level administrators or billing/support roles may need
access to organization-level functions without entering a school.

Define explicit behavior:

-   If the user has organization-level permissions that permit an
    organization dashboard, allow a session with
    `active_school_id = NULL`.
-   If the user has no organization-level access and no active school
    assignments, deny application access and show an appropriate
    account-support or invitation state.
-   Never silently grant access to all schools just because the user has
    no school assignment.

### 5.4 School switching

Example endpoint:

`POST /api/v1/auth/school/switch`

Requirements:

-   Require a valid session.
-   Validate that the target school belongs to the session's
    organization.
-   Validate that the user has an active assignment or an explicit
    organization-wide permission that grants access.
-   Update the active school context server-side.
-   Record a security audit event.
-   Do not trust a school ID merely because it was previously returned
    to the frontend.

A school switch does not change the user's organization.

The session token does not necessarily need to rotate on every school
switch if the session remains the same authenticated principal and the
active context is updated atomically server-side. Rotate or revoke
sessions when credentials, account status, or privileges change in ways
that require it.

## 6. Authorization model

Authorization has three layers.

### 6.1 Organization-level authorization

Organization-level permissions apply to the organization as a whole.

Examples may include:

-   Manage organization settings.
-   Manage organization-wide users and invitations.
-   Create or deactivate schools.
-   View organization-level subscription information.
-   Assign organization-wide administrative permissions.

These permissions must be explicitly granted. They must not be inferred
from the fact that a user belongs to the organization.

### 6.2 School-level authorization

School-level permissions apply only to a specific school.

Examples:

-   View students.
-   Manage student records.
-   Manage classes and enrollment.
-   Manage attendance.
-   Manage assessments.
-   View school reports.
-   Manage school-level staff assignments.

A user may have different school roles in different schools.

Example:

  User     School             Role
  -------- ------------------ ----------------------
  User A   ABC Primary        School administrator
  User A   ABC Secondary      Teacher
  User A   ABC Kindergarten   No assignment

This is why school access should be represented separately from the
global user record.

### 6.3 Role design recommendation

Use explicit organization-level grants and school-level role
assignments.

A practical model is:

-   `users.organization_id`: establishes the user's one organization.
-   `user_schools`: establishes school access and the school-level role.
-   An organization-role assignment or explicit organization permission
    table: establishes organization-wide privileges.

Avoid a single global `users.role` field if users can have different
responsibilities across schools.

Do not assume that an organization administrator automatically needs a
school assignment. If organization administrators should also enter
individual schools, grant those school permissions explicitly or define
a carefully tested organization-wide access rule.

## 7. Session architecture

### 7.1 Opaque server-side sessions

Use opaque random session tokens rather than JWTs.

Recommended properties:

-   Generate at least 256 bits of cryptographically secure random token
    material.
-   Send the raw token only in the browser cookie.
-   Store only a SHA-256 hash of the token in PostgreSQL.
-   Resolve the session on every authenticated request.
-   Check revocation, expiry, user status and organization status.
-   Check school access for school-scoped operations.
-   Never put the raw token in logs, URLs, analytics or client-readable
    storage.

A fast cryptographic hash is appropriate for a high-entropy random
session token. This is different from a six-digit OTP, which requires a
keyed hash because of its small search space.

### 7.2 Cookie configuration

Recommended cookie:

``` http
Set-Cookie: __Host-session=<opaque-token>; Path=/; HttpOnly; Secure; SameSite=Lax
```

Requirements:

-   Use HTTPS.
-   Use the `__Host-` prefix where the deployment topology supports it.
-   Do not set a `Domain` attribute.
-   Set an appropriate `Max-Age` or `Expires`.
-   Keep the token inaccessible to browser JavaScript.
-   Ensure logout clears the cookie using matching cookie attributes.

If the frontend and API deployment topology requires a different
SameSite policy, document the cross-site implications and strengthen
CSRF controls accordingly.

### 7.3 Session expiry

Use both idle and absolute expiry.

Suggested initial values, subject to product and security review:

  Policy              Initial value
  ------------------- ---------------
  Idle timeout        7 days
  Absolute lifetime   30 days

Track:

-   `created_at`
-   `last_seen_at`
-   `expires_at`
-   `revoked_at`
-   `revocation_reason`

Update `last_seen_at` periodically rather than writing on every request.

### 7.4 Session validation

For each authenticated request, Go should:

1.  Read the opaque cookie.
2.  Hash the token.
3.  Find the matching session.
4.  Check session expiry and revocation.
5.  Check user status.
6.  Check organization status.
7.  Resolve the session's organization from trusted database state.
8.  For school-scoped requests, validate the active school and current
    school assignment.
9.  Establish trusted database context for the request transaction.
10. Execute the authorized operation.

### 7.5 Revocation

Support:

-   Current-session logout.
-   Revocation of all sessions for a user.
-   Revocation when a user is suspended.
-   Revocation when a user is removed from the organization.
-   Revocation when an organization is suspended.
-   Revocation or revalidation when security-sensitive permissions
    change.

Because sessions are checked server-side on every request, revocation
can take effect immediately, subject to any carefully controlled
caching.

### 7.6 Organization and school context in a session

Recommended session fields:

-   `user_id`
-   `organization_id`
-   `active_school_id` (nullable)
-   `token_hash`
-   `created_at`
-   `last_seen_at`
-   `expires_at`
-   `revoked_at`
-   `revocation_reason`

`organization_id` is stored for efficient scoping and auditing, but it
must be checked against the user's actual organization. It must never
permit a user to switch organizations.

## 8. PostgreSQL data model

The following is a logical model, not a final migration.

### 8.1 `organizations`

Suggested fields:

-   `id`
-   `name`
-   `status`
-   `created_at`
-   `updated_at`
-   `deleted_at` if soft deletion is required

### 8.2 `users`

Suggested fields:

-   `id`
-   `organization_id` (NOT NULL)
-   `email`
-   `normalized_email`
-   `status`
-   `email_verified_at`
-   `created_at`
-   `updated_at`
-   `deleted_at` if soft deletion is required

Constraints:

-   Unique normalized email, according to the chosen global identity
    policy.
-   Foreign key from `organization_id` to `organizations`.
-   Explicit status values such as `active`, `suspended`, and `revoked`.

### 8.3 `schools`

Suggested fields:

-   `id`
-   `organization_id` (NOT NULL)
-   `name`
-   `status`
-   `created_at`
-   `updated_at`
-   `deleted_at` if soft deletion is required

Constraints:

-   Unique `(organization_id, id)` to support composite foreign keys.
-   Organization foreign key.
-   School status must be checked during authorization.

### 8.4 `user_schools`

Suggested fields:

-   `id`
-   `organization_id`
-   `user_id`
-   `school_id`
-   `role` or `role_id`
-   `status`
-   `created_at`
-   `updated_at`
-   `revoked_at`

Constraints:

-   Unique `(user_id, school_id)`.
-   Composite foreign key `(organization_id, user_id)` to
    `users(organization_id, id)`.
-   Composite foreign key `(organization_id, school_id)` to
    `schools(organization_id, id)`.
-   Indexes for user lookup and school lookup.

These composite constraints prevent an assignment from connecting a user
in one organization to a school in another organization.

### 8.5 Organization-level permissions

If organization-wide roles are required, use a separate table, for
example:

`user_organization_roles`

Suggested fields:

-   `id`
-   `organization_id`
-   `user_id`
-   `role` or `role_id`
-   `status`
-   `created_at`
-   `updated_at`

Enforce the composite relationship between `(organization_id, user_id)`
and the user's organization.

If each user can have only one organization-level role, enforce that
explicitly. If multiple grants are needed, model permissions as grants
rather than adding ambiguous role columns to `users`.

### 8.6 `login_challenges`

Suggested fields:

-   `id`
-   `user_id` (nullable before account resolution, if generic request
    handling requires it)
-   `email_normalized`
-   `purpose`
-   `code_mac`
-   `expires_at`
-   `attempt_count`
-   `max_attempts`
-   `used_at`
-   `invalidated_at`
-   `created_at`

Do not store the plaintext OTP.

### 8.7 `sessions`

Suggested fields:

-   `id`
-   `token_hash`
-   `user_id`
-   `organization_id`
-   `active_school_id` (nullable)
-   `created_at`
-   `last_seen_at`
-   `expires_at`
-   `revoked_at`
-   `revocation_reason`

Use composite foreign keys where practical to ensure the session's user
and school belong to its organization.

### 8.8 `invitations`

Suggested fields:

-   `id`
-   `organization_id`
-   `email_normalized`
-   `token_hash`
-   `scope` (organization or school)
-   `school_id` (nullable)
-   `organization_role` or organization permission grant, if applicable
-   `school_role` or role ID, if applicable
-   `expires_at`
-   `accepted_at`
-   `revoked_at`
-   `created_by_user_id`
-   `created_at`

Only store a hash of the random invitation token.

An invitation may grant organization-level access, school-level access,
or both, depending on the product's invitation policy. The accepted
user's organization must match the invitation's organization. An
existing user from another organization must not be silently moved.

### 8.9 `email_outbox`

Suggested fields:

-   `id`
-   `challenge_id` or invitation reference
-   `message_type`
-   `provider`
-   `status`
-   `attempts`
-   `next_attempt_at`
-   `provider_message_id`
-   `encrypted_payload`
-   `created_at`
-   `sent_at`
-   `last_error_category`

The challenge and outbox record must be committed in the same database
transaction.

Because the email worker needs the OTP in plaintext to send it, the
outbox payload must be encrypted using a key held outside the database.
Remove the encrypted OTP payload after successful delivery and enforce a
short retention period for failed jobs.

Never log decrypted payloads.

### 8.10 `audit_logs`

Suggested fields:

-   `id`
-   `organization_id` (nullable for events before organization
    resolution, where necessary)
-   `actor_user_id` (nullable for unauthenticated events)
-   `event_type`
-   `target_type`
-   `target_id`
-   `school_id` (nullable)
-   `result`
-   `reason_category`
-   `request_id`
-   `ip_metadata` subject to privacy policy
-   `user_agent_metadata` subject to privacy policy
-   `created_at`

Restrict modification and deletion permissions. Define retention and
access policies.

## 9. Database constraints and Row-Level Security

### 9.1 Defense in depth

Use both Go authorization and PostgreSQL Row-Level Security (RLS).

Go must authenticate the user and authorize the requested action.
PostgreSQL RLS must independently restrict access to organization-owned
rows.

### 9.2 Transaction context

After validating the session, Go should establish trusted
transaction-local context, for example:

-   `app.organization_id`
-   `app.user_id`
-   `app.active_school_id`

Use transaction-local settings rather than connection-wide state so
pooled connections cannot leak context between requests.

The values must be set by trusted backend code after session validation,
never copied directly from untrusted request fields.

### 9.3 RLS policy principles

-   Organization-owned records must be restricted by `organization_id`.
-   School-owned records must also be restricted by school
    authorization.
-   School-level access must not be inferred from organization
    membership alone.
-   The application database role must not have `BYPASSRLS`.
-   Review table ownership and `FORCE ROW LEVEL SECURITY` behavior.
-   Test connection-pool reuse and transaction rollback behavior.
-   Include negative tests for missing or malformed context.

RLS is defense in depth, not a replacement for application-level
permission checks.

## 10. CSRF, CORS and browser security

Because authentication uses cookies, implement explicit CSRF defenses.

Recommended controls:

-   `SameSite=Lax` where deployment topology permits.
-   CSRF token validation for state-changing browser requests.
-   Strict Origin validation against known frontend origins.
-   Fetch Metadata validation where supported.
-   Strict CORS allowlist; do not use wildcard origins with credentials.
-   HTTPS throughout.
-   No authenticated response caching by shared caches.
-   Appropriate `Cache-Control: no-store` for authentication and
    sensitive account responses.
-   Login CSRF protection.
-   CSRF protection for logout, school switching, invitations and
    account changes.

Do not treat SameSite cookies as the only CSRF defense.

## 11. Turnstile and rate limiting

Turnstile must be verified server-side using Cloudflare Siteverify.

Validate the successful result and the expected hostname/action where
configured. Treat tokens as short-lived and single-use.

Use independent limits for:

-   Normalized email address.
-   Client IP.
-   IP subnet where justified.
-   Challenge ID.
-   Resend attempts.
-   Global request volume.

Use Redis or another shared, atomic mechanism for distributed rate
limits. Do not rely on per-process in-memory counters when the Go
service has multiple instances.

Only trust forwarded IP headers from known proxy infrastructure.
Configure the origin so clients cannot spoof the source IP used by rate
limiting.

Return generic public errors and record detailed internal reason
categories without logging secrets.

## 12. Email delivery architecture

### 12.1 Provider abstraction

Keep a provider interface such as:

``` go
type CodeSender interface {
    SendCode(ctx context.Context, email string, code string) error
}
```

The interface should not leak provider-specific behavior into
authentication logic.

### 12.2 Transactional outbox

Within one transaction:

1.  Create the login challenge.
2.  Create the encrypted email-outbox payload.
3.  Commit both.

A worker claims pending jobs, decrypts the payload, sends the message,
and records the result.

Use bounded retries with exponential backoff and jitter. Move
persistently failing jobs to a dead-letter state for inspection.

### 12.3 Provider failover

A primary and secondary provider may be configured.

Handle ambiguous timeouts carefully: a provider may have accepted a
message even when the client did not receive its response. Use provider
idempotency support where available and track submission state to avoid
uncontrolled duplicate emails.

Monitor:

-   Submission success.
-   Delivery and bounce signals where available.
-   Provider errors.
-   Queue age.
-   Retry counts.
-   Dead-letter volume.

### 12.4 Brevo email considerations

Brevo is the transactional email provider for OTP and invitation emails,
used behind the provider interface (`CodeSender`). Brevo's service
terms, operational guarantees, recipient restrictions, processing
locations and data-processing terms must be verified before production
reliance.

Configure Brevo with the exact SPF, DKIM and DMARC records supplied
during domain onboarding. Use a dedicated sending subdomain where
practical. Test delivery to real school-managed mailboxes and common
consumer providers.

Brevo API credentials must be held outside the repository (Doppler or
managed secret storage), never in code or config files. The email worker
must never log raw Brevo API keys, response bodies containing PII, or
decrypted outbox payloads.

Email messages should contain the code, expiry guidance, and a warning
not to share the code. Avoid tracking pixels and unnecessary links.

Monitor Brevo submission success, bounce signals, queue age, retry
counts and dead-letter volume. Configure a primary/secondary failover
only if Brevo's SLA requires it; handle ambiguous provider timeouts
using idempotency keys to avoid duplicate emails.

## 13. Invitations and account lifecycle

### 13.1 Invitation creation

An authorized organization administrator:

1.  Chooses the recipient email.
2.  Chooses the invitation scope.
3.  Selects the applicable organization-level permissions and/or school
    assignment and role.
4.  Go validates the inviter's permission to grant those permissions.
5.  Go creates a high-entropy random invitation token.
6.  Only the token hash is stored.
7.  Go creates the email outbox record transactionally.
8.  The invitation is sent and expires according to policy.

An inviter must not be able to grant privileges they are not authorized
to delegate.

### 13.2 Invitation acceptance

1.  Recipient opens the invitation flow.
2.  Recipient authenticates using email OTP.
3.  Go verifies that the authenticated email matches the invitation
    email.
4.  Go checks invitation expiry, revocation and prior acceptance.
5.  Go checks the invitation organization.
6.  If the user already exists, Go checks that their organization
    matches.
7.  Go creates or updates the appropriate organization-level grant
    and/or school assignment.
8.  Go consumes the invitation atomically.
9.  Go records an audit event.

A user who already belongs to another organization must not be silently
reassigned. The product needs an explicit support or
organization-transfer process if transfers are ever permitted.

### 13.3 Membership and school assignment lifecycle

Define behavior for:

-   User suspension.
-   User removal from an organization.
-   School assignment revocation.
-   School suspension.
-   Organization suspension.
-   Invitation expiry and revocation.
-   Email address changes.
-   Organization transfer, if ever introduced.

A revoked school assignment must prevent access to that school on the
next authorized request, even if the browser still holds a valid
session.

An organization suspension must prevent access to all organization data.

## 14. Account recovery and email changes

The account recovery process must not become a weaker route around OTP
authentication.

Define:

-   What happens when a user loses access to their email.
-   Whether an organization administrator can initiate a recovery
    process.
-   What independent verification is required.
-   How the old email is invalidated.
-   Whether all sessions are revoked after an email change.
-   How the change is audited.
-   How duplicate or conflicting email identities are handled.

An email change should require re-verification and should not silently
change the user's organization.

Do not introduce SMS or another channel without assessing its security,
cost, delivery reliability, and applicable privacy requirements.

## 15. API surface

The following is a suggested API surface. Exact names may be adjusted to
existing conventions, but responsibilities should remain separate.

  ---------------------------------------------------------------------------------------
  Method                  Endpoint                                Purpose
  ----------------------- --------------------------------------- -----------------------
  POST                    `/api/v1/auth/otp/request`              Request login OTP

  POST                    `/api/v1/auth/otp/resend`               Resend OTP

  POST                    `/api/v1/auth/otp/verify`               Verify OTP and
                                                                  establish identity

  GET                     `/api/v1/auth/me`                       Return authenticated
                                                                  user and organization
                                                                  context

  GET                     `/api/v1/auth/schools`                  List schools the user
                                                                  may access

  POST                    `/api/v1/auth/school/select`            Select initial school
                                                                  context

  POST                    `/api/v1/auth/school/switch`            Switch active school

  POST                    `/api/v1/auth/logout`                   Revoke current session

  POST                    `/api/v1/auth/logout-all`               Revoke all user
                                                                  sessions

  POST                    `/api/v1/invitations`                   Create invitation

  POST                    `/api/v1/invitations/accept`            Accept invitation

  POST                    `/api/v1/account/email/change`          Begin email change

  POST                    `/api/v1/account/email/change/verify`   Verify new email
  ---------------------------------------------------------------------------------------

All authenticated endpoints derive the user and organization from the
validated session. School-scoped endpoints validate the requested school
against current access.

## 16. Suggested response context

An authenticated identity response may expose a safe representation such
as:

``` json
{
  "user": {
    "id": "user-uuid",
    "email": "person@example.com"
  },
  "organization": {
    "id": "organization-uuid",
    "name": "ABC Education Group"
  },
  "active_school": {
    "id": "school-uuid",
    "name": "ABC Primary School"
  },
  "available_schools": [
    {
      "id": "school-uuid",
      "name": "ABC Primary School"
    },
    {
      "id": "another-school-uuid",
      "name": "ABC Secondary School"
    }
  ],
  "permissions": {
    "organization": [],
    "school": []
  }
}
```

This is presentation data, not an authorization credential. The backend
must revalidate permissions for every protected operation.

## 17. Logging, audit and monitoring

Record security-relevant events, including:

-   OTP requested, sent, failed and verified.
-   Rate limit triggered.
-   Session created, expired and revoked.
-   Login and logout.
-   School selected or switched.
-   Invitation created, accepted, expired or revoked.
-   User suspended or reactivated.
-   School assignment granted or revoked.
-   Organization suspended.
-   Authorization denied.

Never log:

-   Plaintext OTPs.
-   Raw session tokens.
-   Raw invitation tokens.
-   HMAC or encryption secrets.
-   Decrypted outbox payloads.
-   Full authentication request bodies.

Monitor:

-   OTP request and verification rates.
-   Invalid and expired OTP rates.
-   Rate-limit activity.
-   Email queue age and delivery outcomes.
-   Session creation and revocation.
-   Repeated authorization denials.
-   Cross-organization access attempts.
-   RLS denials and database errors.

Define retention, access control and deletion policies for logs,
including IP and user-agent metadata.

## 18. Privacy and regulatory review

Before production use with real school or student data, obtain qualified
Kenyan data-protection advice.

Review at least:

-   ODPC registration obligations.
-   Controller and processor roles.
-   Children's personal data and parental consent requirements.
-   International data transfers.
-   Cloudflare and email-provider processing locations.
-   Hosting, backup, logging and monitoring locations.
-   Data processing agreements and subprocessors.
-   Data subject rights.
-   Retention and deletion.
-   Security incident and breach notification obligations.

Assess the full data flow, not only the PostgreSQL hosting region. Email
addresses, IP metadata, authentication events and support access may
also involve personal data.

Document the data flow and approved processing locations with the legal
review.

## 19. Security test requirements

### P0 --- Required before production

-   OTP expires after the configured validity period.
-   OTP cannot be reused.
-   OTP verification cannot exceed the attempt limit.
-   Concurrent OTP verification cannot consume the same challenge twice.
-   OTP database leakage alone does not enable practical offline
    guessing.
-   Resending invalidates the prior active challenge.
-   Turnstile invalid, expired and replayed tokens are rejected.
-   Account existence is not disclosed through OTP request responses.
-   Session tokens are not stored in plaintext.
-   Expired sessions are rejected.
-   Revoked sessions are rejected immediately.
-   Suspended users cannot authenticate or continue using existing
    sessions.
-   Suspended organizations cannot be accessed.
-   A user cannot select a school outside their organization.
-   A user cannot select a school without an active assignment or
    explicit organization-wide permission.
-   A user cannot access another organization's records by changing
    request parameters.
-   Composite foreign keys prevent cross-organization school
    assignments.
-   RLS denies access when organization context is missing or incorrect.
-   RLS remains correct under connection-pool reuse.
-   CSRF requests from untrusted origins are rejected.
-   Spoofed forwarded IP headers cannot bypass rate limits.
-   Authentication secrets never appear in logs.

### P1 --- Required before broad rollout

-   Invitation tokens expire and cannot be reused.
-   Invitation email must match the authenticated email.
-   Users cannot accept invitations into a second organization.
-   Inviter cannot grant privileges they cannot delegate.
-   School assignment revocation takes effect immediately.
-   Organization-level permissions are tested separately from school
    roles.
-   Email retries handle ambiguous provider timeouts safely.
-   Dead-letter email jobs are observable and recoverable.
-   All sessions can be revoked for a user.
-   Email changes require verification and produce audit events.
-   Audit events are attributable and access-controlled.

## 20. Implementation sequence

Implement in dependency order.

### Phase 1 --- Identity and authorization foundations

1.  Finalize organization, user, school and school-assignment model.
2.  Finalize organization-level and school-level role semantics.
3.  Add unique constraints and composite foreign keys.
4.  Add organization and school authorization helpers.
5.  Add PostgreSQL RLS policies and integration tests.

### Phase 2 --- Authentication

1.  Implement OTP generation and HMAC storage.
2.  Implement Turnstile verification.
3.  Implement distributed rate limiting.
4.  Implement atomic OTP verification.
5.  Implement the transactional email outbox and worker.
6.  Implement generic authentication responses.

### Phase 3 --- Sessions and school context

1.  Implement opaque session tokens.
2.  Implement secure cookie issuance and clearing.
3.  Implement session validation and revocation.
4.  Implement authenticated identity endpoint.
5.  Implement school listing and selection.
6.  Implement school switching.
7.  Implement CSRF and CORS controls.

### Phase 4 --- Organization operations

1.  Implement invitations.
2.  Implement school assignments and role grants.
3.  Implement suspension and revocation workflows.
4.  Implement account recovery and email changes.
5.  Implement audit events and monitoring.

### Phase 5 --- Production readiness

1.  Complete security review.
2.  Complete Kenyan privacy and data-protection review.
3.  Verify email provider terms, reliability and processing locations.
4.  Test real school email delivery.
5.  Run cross-organization and RLS penetration tests.
6.  Validate backup, restore and incident-response procedures.
7.  Confirm operational alerts and on-call ownership.

## 21. Decisions to confirm before implementation

The architecture is ready for detailed API and schema design once these
product decisions are confirmed:

1.  **Organization-level roles:** Which organization-wide roles exist,
    and which actions do they permit?
2.  **School-level roles:** Can a user have a different role in each
    school? This specification recommends yes.
3.  **Organization administrators:** Can they access organization-level
    functions without being assigned to a school? This specification
    recommends yes, when explicitly authorized.
4.  **Organization-wide school access:** Should any role automatically
    access every school, or must access be explicitly assigned? The
    safer default is explicit assignment, with a clearly defined
    exception if needed.
5.  **Invitations:** Can an invitation grant organization-level access,
    school-level access, or both?
6.  **Email uniqueness:** Confirm that one normalized email address
    represents one global user identity.
7.  **Recovery:** Define the approved process when a user loses access
    to their email.
8.  **Session policy:** Confirm idle and absolute expiry values.

## 22. Non-negotiable invariants

These statements should become architecture tests and code-review
checks:

1.  A user belongs to exactly one organization.
2.  A school belongs to exactly one organization.
3.  A user can access only schools within their organization.
4.  A user may have different roles in different schools.
5.  Organization-level permissions and school-level permissions are
    distinct.
6.  The organization is derived from trusted server-side
    identity/session data.
7.  The active school is validated by Go for every school-scoped
    operation.
8.  The frontend never makes authoritative authorization decisions.
9.  PostgreSQL RLS provides a second layer of organization isolation.
10. OTPs, session tokens and invitation tokens are never stored in
    plaintext.
11. Session revocation and membership/assignment revocation take effect
    on subsequent requests.
12. No organization or school boundary may be bypassed by changing
    client-supplied IDs.

------------------------------------------------------------------------

**End of specification.**

This document is the architecture baseline. Database migrations, API
contracts, sequence diagrams, threat models and implementation tickets
should be derived from it rather than introducing conflicting rules.
