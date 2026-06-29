# GET /api/auth/invite/callback

**Group:** Auth

**Auth:** None

**Description:** Invitation acceptance callback. Validates an invitation token from query params, completes the user registration if needed, and sets up the user's membership in the tenant/school. Redirects to the frontend on success.

**Query params:**

- `token` (string, required) - The invitation token
- `redirect` (string, optional) - Frontend redirect URL

**Response `302`:** Redirect to frontend with session cookie set.
