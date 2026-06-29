# GET /api/v1/imports/students/stream

**Group:** Imports - Students

**Auth:** Session cookie (`somo_sid`) required

**Description:** Server-Sent Events endpoint that streams real-time progress updates for the most recent student import (or a specific import via query param). Opens a long-lived connection and pushes status events.

**Query params:**

- `import_id` (string, optional) - The import UUID to track

**Response:** SSE stream

```
data: {"import_id":"uuid","status":"PROCESSING","total":50,"processed":25,"failed":1}
data: {"import_id":"uuid","status":"COMPLETED","total":50,"processed":49,"failed":1}
```
