# POST /tenants

**Group:** Tenant

**Auth:** None

**Description:** Creates a new tenant (educational institution / school district) in the system. This is the top-level organisational unit. Returns the created tenant with its UUID and slug.

**Request body:**

```json
{
  "name": "string (required)",
  "slug": "string (optional - auto-generated if omitted)"
}
```

**Response `201`:**

```json
{
  "id": "uuid",
  "name": "string",
  "slug": "string"
}
```
