# SPEC-0XX: [Feature Title]

Endpoint: `METHOD /api/v1/...`

## Metadata

| Field | Value |
|-------|-------|
| id | SPEC-0XX |
| title | [Feature Title] |
| status | draft |
| depends_on | specs/glossary.md, specs/001-auth-otp-request.md, ADR-0XXX |
| api_endpoint | METHOD /api/v1/... |
| owner | backend | frontend | fullstack |

## Summary

One paragraph describing what this feature does and how it fits into the system.

## Request

```
METHOD /api/v1/...
Content-Type: application/json
Cookie: (if applicable)

{
  "field1": "value",
  "field2": "value"
}
```

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| field1 | string | Yes | Description |
| field2 | string | No | Description |

## Response (success)

```
HTTP 200 OK
Content-Type: application/json
Set-Cookie: (if applicable)

{
  "ok": true,
  "message": "...",
  ...
}
```

## Response (errors)

| Status | Code | Condition |
|--------|------|-----------|
| 400 | `INVALID_REQUEST` | Malformed JSON or missing fields |
| 401 | `UNAUTHENTICATED` | No or invalid session |
| 403 | `FORBIDDEN` | Insufficient permissions |
| 404 | `NOT_FOUND` | Resource not found |
| 429 | `RATE_LIMITED` | Too many requests |
| 500 | `INTERNAL_ERROR` | Unexpected server error |

## Business rules

1. [Rule 1]
2. [Rule 2]
3. [Rule 3]

## Tests required

| Test type | What to verify |
|-----------|----------------|
| Unit | [Description] |
| Integration | [Description] |
| Contract | [Description] |

## Acceptance criteria

Given [context]
When [action]
Then [expected outcome]

## Open questions

- [Question 1]
- [Question 2]

## Links

- Architecture: [link]
- Glossary: `specs/glossary.md`
