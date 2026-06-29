# GET /api/auth/callback

**Group:** Auth

**Auth:** None

**Description:** Magic link callback handler. Accepts a token query parameter sent via email magic link, validates it, and establishes a session. Redirects the user to the frontend on success.

**Query params:**

- `token` (string, required) - The magic link token
- `redirect` (string, optional) - Frontend redirect URL

**Response `302`:** Redirect to frontend with session cookie set.
