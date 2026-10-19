# Gap: Class Timetable Slots Endpoint Missing Class Ownership Validation (BOLA)

**Area:** backend
**Severity:** Critical
**Status:** Open
**Owner:** 
**Created:** 2025-10-19
**Related Issues:** #security-audit-2025-10-19

## Description
The `GET /api/timetable/templates/:id/classes/:classId/slots` endpoint accepts a `classId` path parameter but does not verify that the class belongs to the requesting user's school. This allows cross-school data access (BOLA).

## Current Behavior
In `timetable_handler.go:145-165`:
```go
func (h *TimetableHandler) GetClassSlotsByTemplate(c fiber.Ctx) error {
    schoolIDStr, ok := c.Locals("active_school_id").(string)
    // ... parse schoolID ...
    templateIDStr := c.Params("id")
    classIDStr := c.Params("classId")
    templateID, err := uuid.Parse(templateIDStr)
    classID, err := uuid.Parse(classIDStr)
    // ...
    rows, err := h.svc.GetClassTimetableSlotsByTemplate(c.Context(), schoolID, classID, templateID)
    // ...
}
```
`classID` passed directly to service without verifying `classID` → `schoolID` ownership.

## Expected Behavior
Service must validate that `classID` belongs to `schoolID` before returning slots. Return 404 if not.

## Impact
- **Data Leak**: User in School A can enumerate slots for classes in School B by guessing class IDs
- **Reconnaissance**: Allows mapping other schools' timetable structures
- **Compliance**: OWASP API1:2023 BOLA — Critical severity

## Reproduction Steps
1. School A has class `class-uuid-1` with timetable slots
2. User in School B calls `GET /api/timetable/templates/template-uuid/classes/class-uuid-1/slots`
3. Observe slots returned for School A's class

## Proposed Solutions
- **Option A (Service validation)**: Service `GetClassTimetableSlotsByTemplate` checks class ownership via DB query
- **Option B (Handler validation)**: Handler calls `classesSvc.GetClass(ctx, schoolID, classID)` first (404 if not found)
- **Option C (RLS)**: PostgreSQL RLS policy on `class_timetable_slots` with `school_id` join

## Decision
Pending — recommend Option A (service validation) + Option C (RLS).

## Acceptance Criteria
- [ ] Service validates `classID` belongs to `schoolID` before querying slots
- [ ] Returns 404 (not 403) if class not in school (prevents enumeration)
- [ ] Integration test: User A cannot access User B's class slots → 404
- [ ] RLS policy verified on relevant tables

## Links
- Security audit report: `security-report-extended.md`
- OWASP API1:2023 reference: `docs/reference/owasp-api-top10-2023.md#api12023---broken-object-level-authorization-bola`
- Handler: `backend/internal/api/timetable_handler.go:145`
- Service: `backend/internal/services/timetable_service.go` (GetClassTimetableSlotsByTemplate)