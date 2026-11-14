---
id: SOM-PAGINATION-001
title: Backend Keyset Pagination & Cursor Engine
status: draft
author: agent
date: 2026-11-14
updated: 2026-11-14
fr_ids: [BE-006, BE-007, BE-009, BE-010, BE-012, BE-014, BE-015, BE-016, BE-017, BE-020, BE-021, BE-023, BE-024, BE-025, BE-035, BE-036, FE-007, FE-008, FE-011, FE-019, FE-026]
nf_ids: [BE-031, BE-032]
api_contract: no
migrations: no
generated_files: []
---

## 1. Summary

A reusable Go query engine for keyset pagination at `backend/internal/platform/cursor/`. It provides cursor encoding/decoding, validation, and query parameter parsing for backend list endpoints. This spec is the backend counterpart of `SOM-DATATABLE-001` (frontend data table core) — the two must stay aligned on the cursor wire format.

The engine does **not** generate SQL. Per BE-009, all SQL lives in `store/queries/*.sql` compiled by sqlc. Each domain declares whitelisted sorts with one named sqlc query per sort containing a hardcoded keyset predicate. The cursor package only:

1. Parses and validates query parameters (`limit`, `cursor`, `sort`, `filters`)
2. Encodes/decodes HMAC-signed Base64url JSON cursors
3. Validates cursor context hash (sort + filter binding)
4. Returns typed errors (`INVALID_CURSOR`, `BAD_REQUEST`)

This aligns `BE-014` (keyset pagination: `limit`/`cursor`/`sort`, `{items,nextCursor}`, limit default 50 max 100, no OFFSET/total count) with `BE-023` ("Cursor pages use one generic wrapper") and `SOM-DATATABLE-001` (frontend cursor contract).

## 2. Requirements (FR/NFR IDs with rationale)

| FR/NFR ID | Rationale |
|-----------|-----------|
| BE-006 | Handlers remain thin: bind query params, call service, map to response; cursor parsing lives in the engine |
| BE-007 | Handlers return named DTOs; engine returns structured params + cursor token |
| BE-009 | SQL lives only in `store/queries/*.sql` via sqlc; engine generates no SQL |
| BE-010 | Stores map sqlc rows to domain types; engine holds no business logic |
| BE-012 | One transaction per service call; engine is read-only and transaction-agnostic |
| BE-014 | Keyset pagination: `limit`/`cursor`/`sort`, `{items,nextCursor}`, limit default 50 max 100, fetch `limit+1`, UUID tiebreaker, no OFFSET/total count |
| BE-015 | DB invariants (constraints/indexes) support keyset predicates; engine never bypasses them |
| BE-016 | Tenant scoping applied via session context + RLS; cross-tenant cursor misuse returns 404 |
| BE-017 | Tenant ID comes only from session context (middleware), never from client query params |
| BE-020 | Route authorization (`authz.Require`) is orthogonal; engine does not perform authz |
| BE-021 | `ScopeFor` provides tenant scope; engine does not replace it |
| BE-023 | One generic cursor wrapper for all list pages |
| BE-024 | Validation tags on request DTOs; engine validates query params and maps failures to stable `errors[].code` |
| BE-025 | SCREAMING_SNAKE error codes (`INVALID_CURSOR`), camelCase query params |
| BE-035 | Reference FR/NFR IDs; raise Open Questions instead of assuming |
| BE-036 | Unit + integration tests with real PostgreSQL (testcontainers); cross-tenant isolation |
| FE-007 | Frontend never sends school/tenant ID; cursor is opaque |
| FE-008 | Generated API types; cursor is a string in `ListStudentsResponse` etc. |
| FE-011 | Active school from backend state, never from cursor |
| FE-019 | Cursor pagination on frontend (`useInfiniteQuery`, `nextCursor`) |
| FE-026 | FR/NFR IDs referenced; Given/When/Then acceptance criteria |

## 3. Domain model affected

No database tables, migrations, or schema changes. This is a platform package.

### `ListConfig` (declarative configuration)

Defined in `backend/internal/platform/cursor/config.go`. One `ListConfig` per list endpoint, declared by the owning domain.

```go
type ListConfig struct {
    // DefaultSort is the default sort name (e.g. "name_asc").
    DefaultSort string

    // DefaultLimit is the default page size (BE-014: 50).
    DefaultLimit int

    // MaxLimit is the maximum allowed page size (BE-014: 100).
    MaxLimit int

    // SortFields declares whitelisted sorts. Each sort name maps to a sqlc query
    // with a hardcoded keyset predicate.
    SortFields []SortField

    // FilterFields declares whitelisted filter fields and allowed operators.
    FilterFields []FilterField
}

type SortField struct {
    // Name is the sort value accepted in the `sort` query param (e.g. "name_asc").
    Name string

    // Columns is the composite sort key (1-3 columns).
    Columns []SortColumn
}

type SortColumn struct {
    Column  string // DB column name
    Asc     bool   // sort direction
    NullsLast bool   // NULLS LAST handling
}

type FilterField struct {
    // Name is the query param name (camelCase, e.g. "fullName").
    Name string

    // Column is the DB column name (snake_case, e.g. "full_name").
    Column string

    // Operators is the allowlist of operators for this field.
    Operators []Operator
}

type Operator string

const (
    OpEq      Operator = "eq"      // =
    OpIn      Operator = "in"      // IN (...)
    OpBetween Operator = "between" // BETWEEN ... AND ...
    OpContains Operator = "contains" // ILIKE %...%
    OpIsNull  Operator = "is_null" // IS NULL / IS NOT NULL
)
```

Config validation at init:
- Max 3 sort columns per sort
- Unique sort names
- Unique filter names
- `DefaultLimit` <= `MaxLimit`, both >= 1

### Parsed query parameters

```go
type ListParams struct {
    Limit  int
    Sort   string          // resolved sort name
    Cursor *Cursor         // decoded cursor, or nil for first page
    Filters map[string]FilterValue // param name -> value + operator
}
```

### HMAC-signed cursor

The cursor is an opaque string to the client. The frontend never constructs one.

```go
type cursorPayload struct {
    Sort   []string      `json:"s"`  // sort names bound to this cursor
    Last   []interface{} `json:"l"`  // last row values for each sort column
    Tiebreaker string   `json:"t"`  // UUID PK of last row
    Ctx    string        `json:"c"`  // query-context hash
}
```

Encoded as Base64url(JSON) with no HMAC signature (cursor is opaque; tampering yields decode error). Context hash:

```
ctxHash = sha256(sortName + "|" + filtersEncoded)
```

where `filtersEncoded` is the exact Base64url `filters` query param value.

### `WhereFragment` output (for documentation only; not generated by engine)

The engine does not generate SQL fragments. Each domain's sqlc query hardcodes the keyset predicate:

```sql
-- Example: students list, sort name_asc
SELECT * FROM students
WHERE tenant_id = current_tenant_id()
  AND deleted_at IS NULL
  AND ($1::text IS NULL OR full_name ILIKE '%' || $1 || '%')
  AND (full_name, student_id) > ($2, $3)  -- keyset predicate
ORDER BY full_name ASC NULLS LAST, student_id ASC
LIMIT $4
```

The `$2, $3` values come from the decoded cursor's `Last` and `Tiebreaker` fields.

### Business rules

- `limit` defaults to 50; maximum 100; values above 100 → `BAD_REQUEST` (400).
- Non-numeric `limit`, unknown `sort`, unknown filter field, or disallowed operator → `BAD_REQUEST` (400).
- Cursor is bound to the exact sort + filter set (context hash). A cursor from a different sort/filter combination → `INVALID_CURSOR` (400).
- Tampered or malformed cursor → `INVALID_CURSOR` (400).
- Cursor never contains tenant/school IDs; tenant scope is applied server-side from session context (BE-017).
- Fetch `limit + 1` rows; if `limit + 1` returned, `nextCursor` is set from the last row's sort values + UUID tie-breaker.
- No OFFSET, no total count (per BE-014).
- Deactivated/soft-deleted rows (`deleted_at IS NULL`) are excluded by the domain's sqlc query, not the engine.

## 4. API contract

No new endpoints. The cursor remains an opaque `string` query parameter already documented in OpenAPI (e.g. `ListStudentsResponse.nextCursor`). This spec governs *how* backends generate and validate that cursor.

Error code `INVALID_CURSOR` (400) must be added to `backend/internal/platform/apperr/errors.go` and registered with status 400.

No OpenAPI change required; `nextCursor` is already `string | null` in list responses.

Frontend (`SOM-DATATABLE-001`) sends filters as `filters=<base64url>` in the query string. The backend parser must match this serialization exactly.

## 5. Workflow (Given/When/Then acceptance criteria)

### Query parameter parsing

**Given** `GET /api/v1/students?limit=50&sort=name_asc`
**When** the parser runs
**Then** returns `ListParams{Limit: 50, Sort: "name_asc", Cursor: nil}`

**Given** `limit=abc`
**When** the parser runs
**Then** returns `BAD_REQUEST` (400)

**Given** `sort=unknown_sort`
**When** the parser runs
**Then** returns `BAD_REQUEST` (400)

**Given** `filter[created_at]=eq:2024-01-01` where `created_at` is not in the filter allowlist
**When** the parser runs
**Then** returns `BAD_REQUEST` (400)

**Given** `limit=200` (above MaxLimit 100)
**When** the parser runs
**Then** returns `BAD_REQUEST` (400)

### Cursor encoding/decoding

**Given** a cursor payload produced by the engine
**When** encoding then decoding
**Then** returns the original payload

**Given** a cursor with a single byte flipped in the Base64url payload
**When** decoding
**Then** returns `INVALID_CURSOR` (400)

**Given** a cursor from sort `name_asc` used on a request with sort `created_at_desc`
**When** verifying the context hash
**Then** returns `INVALID_CURSOR` (400) — sort/filter context mismatch

**Given** a cursor from tenant A used on a request from tenant B
**When** the service applies tenant scope
**Then** the tenant-scoped query returns no matching row → 404 (cursor is bound to sort+filters, tenant enforced server-side)

### Integration: full keyset walk

**Given** 100,000+ seeded rows in a real PostgreSQL (testcontainers)
**When** walking all pages via `nextCursor` until `nextCursor` is null
**Then** every row is fetched exactly once, zero duplicates, zero missing rows

**Given** the walk above with mixed ASC/DESC composite sorts
**Then** ordering is consistent across page boundaries; NULLS LAST respected

## 6. Implementation plan (files to touch, by application)

### Backend — new package `backend/internal/platform/cursor/`

1. **`config.go`** — `ListConfig`, `SortField`, `SortColumn`, `FilterField`, `Operator` types; config validation at init.
2. **`parser.go`** — `ParseListParams(query url.Values, cfg *ListConfig) (*ListParams, error)`; validates limit, sort, filters; rejects SQL injection (opaque text handling).
3. **`cursor.go`** — `EncodeCursor(payload) (string, error)`, `DecodeCursor(token string) (*cursorPayload, error)`, Base64url encoding, context-hash computation (`sha256(sortName + "|" + filtersEncoded)`).
4. **`errors.go`** — engine-specific error wrapping with `apperr` codes (`BAD_REQUEST`, `INVALID_CURSOR`).
5. **`cursor_test.go`** — parser unit tests (table-driven), cursor security tests.
6. **`cursor_integration_test.go`** — 100k-row keyset walk via testcontainers PostgreSQL.

### Backend — apperr

7. **`backend/internal/platform/apperr/errors.go`** — add `INVALID_CURSOR Code = "INVALID_CURSOR"`; register with status 400 in `init()`.

### Backend — domain stores (follow-up, not part of this spec)

Each domain that lists resources declares a `ListConfig` and writes one sqlc query per whitelisted sort with a hardcoded keyset predicate. The store calls `ParseListParams` and `DecodeCursor` to validate input, then passes cursor values as sqlc query parameters.

## 7. Tests required

### Backend (per BE-036)

**Unit tests (`cursor_test.go`):**

Query Parameter Parser (table-driven):
- Valid `limit`, `cursor`, `sort`, `q`, filter params parse correctly
- `limit` below 1 → `BAD_REQUEST`
- `limit` above `MaxLimit` → `BAD_REQUEST`
- Non-numeric `limit` → `BAD_REQUEST`
- Unknown `sort` → `BAD_REQUEST`
- Unknown filter field → `BAD_REQUEST`
- Disallowed operator for a field → `BAD_REQUEST`
- `between` without two values → `BAD_REQUEST`
- `in` with empty list → `BAD_REQUEST`
- `is_null` with non-boolean value → `BAD_REQUEST`
- Filter values treated as opaque text (no SQL interpolation)
- Empty/missing params use defaults (`limit=50`, `sort=DefaultSort`)

Cursor Security:
- Round-trip: encode → decode returns original payload
- Single-byte flip in Base64url payload → `INVALID_CURSOR`
- Corrupted Base64url → `INVALID_CURSOR`
- Context-hash mismatch (cursor from different sort) → `INVALID_CURSOR`
- Context-hash mismatch (cursor from different filters) → `INVALID_CURSOR`

### Integration tests (`cursor_integration_test.go`, testcontainers PostgreSQL)

- Seed 100,000+ rows in a table with columns matching a composite sort
- Walk all pages via `nextCursor` until null
- Assert every row fetched exactly once (zero duplicates, zero missing)
- Assert page ordering matches `ORDER BY` + NULLS LAST
- Assert cross-tenant isolation (tenant-scoped queries)

### apperr registration test

- `INVALID_CURSOR` registered with status 400; `apperr.Status(INVALID_CURSOR) == 400`

## 8. Open questions

1. **`INVALID_CURSOR` code placement:** Is `INVALID_CURSOR` a generic code (apperr) or a pagination-domain code? Recommendation: generic, since pagination is cross-domain. Register in `apperr/errors.go` `init()`.

2. **Filter wire format:** Must match frontend `SOM-DATATABLE-001` exactly: single `filters=<base64url>` param with snake_case field names and `op`/`v` structure. Confirm backend parser matches `createListQueryFn` serialization.

## 9. Links

- [BE-014](../requirements/backend.md) — keyset pagination rules
- [BE-023](../requirements/backend.md) — generic cursor wrapper
- [BE-016](../requirements/backend.md) — tenant-scoped queries
- [BE-017](../requirements/backend.md) — session context, no client scope
- [BE-009](../backend/AGENTS.md) — SQL only in store/queries/*.sql, no SQL in Go strings
- [BE-004](../backend/AGENTS.md) — shared pure helper package (`platform/cursor`)
- [SOM-DATATABLE-001](./SOM-DATATABLE-001.md) — frontend data table core; defines the cursor wire format this spec implements
- [SOM-SCHOOLS-001](./SOM-SCHOOLS-001.md) — draft school spec; will use this cursor engine for `/schools` list
- [backend/internal/platform/apperr/errors.go](../../../backend/internal/platform/apperr/errors.go) — add `INVALID_CURSOR`
- [backend/internal/platform/cursor/](../../../backend/internal/platform/cursor/) — new package (to be created on implementation)
