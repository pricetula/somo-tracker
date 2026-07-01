---
name: generate-api
description: Scaffold a new Fiber API endpoint (service interface + router entry + Postman stub).
trigger: user requests add
---

# Generate API Endpoint

## Output Example

```
POST /api/v1/endpoint
{
  "field": "value"
}
```

## Task Loop
1. Extract domain model and action intent from user request.
2. Create service interface in `internal/service/`.
3. Create or update router entry in `internal/router/`.
4. Generate Postman collection stub.
5. Run `make build-backend` to verify compilation.
