# DELETE /api/v1/classes

**Group:** Classes (CBC)

**Auth:** Session cookie (`somo_sid`) required

**Description:** Bulk deletes multiple classes by their IDs. Max 100 class IDs per request. Will fail with `CLASS_HAS_ASSESSMENTS` if any of the classes have assessment records.

**Request body:**

```json
{
  "class_ids": ["uuid", "uuid", ...]
}
```

**Response `204`:** No content on success.

**Error `409`:**

```json
{
  "error": "CLASS_HAS_ASSESSMENTS",
  "message": "One or more classes have assessment records and cannot be deleted."
}
```
