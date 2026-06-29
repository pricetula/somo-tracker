# DELETE /api/v1/schools/:id

**Group:** Schools (CBC)

**Auth:** Session cookie (`somo_sid`) required

**Description:** Deletes a school. Verifies the school belongs to the authenticated user's tenant before deletion. Returns 204 on success.

**Response `204`:** No content on success.
