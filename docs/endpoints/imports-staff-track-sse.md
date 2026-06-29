# GET /api/v1/imports/staff/track/:id/sse

**Group:** Imports - Staff

**Auth:** Session cookie (`somo_sid`) required

**Description:** Server-Sent Events endpoint that streams real-time progress updates for a staff import. Opens a long-lived connection and pushes status events as the import progresses.

**URL params:**

- `id` (string, required) - The import UUID

**Response:** SSE stream

```
data: {"import_id":"uuid","status":"PROCESSING","total":10,"processed":2,"failed":0}
data: {"import_id":"uuid","status":"COMPLETED","total":10,"processed":10,"failed":0}
```
