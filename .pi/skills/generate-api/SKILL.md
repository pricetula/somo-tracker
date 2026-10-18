---
name: generate-api
description: Scaffold a new Fiber API endpoint (service interface + router entry + Postman stub). Use when user requests adding a new endpoint, route, or API feature.

# Generate API Endpoint

## Output Example
```
POST /api/v1/endpoint
{
  "field": "value"
}
```

## Task Loop

### Step 1 — Extract Domain Model and Action Intent
1. Parse user input to extract:
   - Resource type (e.g., Student, Teacher, Class)
   - Action (create, read, update, delete, list)
   - Domain-specific constraints (tenant isolation, academic period)
2. Determine if the endpoint needs:
   - RLS (Row-Level Security) – if it's tenant-scoped data
   - Academic period resolution – if it's term-bound
   - Rate limiting – if it's a sensitive flow

### Step 2 — Create Service Interface
- Place in `internal/services/<domain>.go`
- Define interface methods with clear naming
- Include error types from `internal/api/errors.go`
- Use sqlc-generated queries only (never raw SQL in handlers)

### Step 3 — Create or Update Router Entry
- Place in `internal/api/<domain>_handler.go`
- Register route in `internal/api/router.go`
- Combine group prefix + route string → full URI
- Verify with `make generate-swagger` after adding

### Step 4 — Generate Postman Collection Stub
- Export Postman collection from swagger.json
- Place in `docs/postman/`
- Include request examples and expected responses

### Step 5 — Run `make build-backend` to Verify Compilation
- Run `make vet` for static analysis
- Run `make test` for unit tests
- Run `make test-integration` for integration tests

## Integration Points
- Uses `backend/AGENTS.md` for error handling and academic period rules
- Uses `backend/internal/api/errors.go` for canonical error contract
- Uses `backend/internal/database/migrator_integration_test.go` for migration testing
- Uses `backend/docs/swagger.json` for API documentation
- Uses `frontend/src/lib/api/client.ts` for frontend integration

## Prerequisites
- `backend/internal/services/` directory exists
- `backend/internal/api/` directory exists
- `backend/internal/api/router.go` exists
- `backend/docs/swagger.json` exists
- `make build-backend` is executable