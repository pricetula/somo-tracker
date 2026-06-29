# DELETE /api/auth/session

**Group:** Auth

**Auth:** Session cookie (`somo_sid`) required

**Description:** Destroys the current user session. Clears the session cookie and invalidates the session token server-side. Used for logout.

**Response `200`:**

```json
{
  "message": "session destroyed"
}
```
