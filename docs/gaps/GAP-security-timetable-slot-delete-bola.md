# Gap: Timetable Slot Delete Missing School Ownership Check (BOLA)

**Area:** backend
**Severity:** Critical
**Status:** Open
**Owner:** 
**Created:** 2025-10-19
**Related Issues:** #security-audit-2025-10-19

## Description
The `DELETE /api/timetable/class-timetable-slots/:id` endpoint deletes a timetable slot by ID only, without verifying the slot belongs to the requesting user's school. This allows cross-school deletion (BOLA - Broken Object Level Authorization).

## Current Behavior
In `timetable_handler.go:168-180`:
```go
func (h *TimetableHandler) DeleteClassTimetableSlot(c fiber.Ctx) error {
    slotIDStr := c.Params("id")
    slotID, err := uuid.Parse(slotIDStr)
    if err != nil {
        return WriteError(c, ErrBadRequest("invalid slot id", nil))
    }
    if err := h.svc.DeleteClassTimetableSlot(c.Context(), slotID); err != nil {
        // ...
    }
    return c.Status(fiber.StatusNoContent).Send(nil)
}
```
Only `slotID` passed to service. No `schoolID` from context.

## Expected Behavior
Handler must extract `active_school_id` from `c.Locals()`, pass to service. Service must verify slot belongs to that school before deletion.

## Impact
- **Data Loss**: User in School A can delete slots in School B by guessing IDs
- **Integrity**: Timetable data corrupted across tenants
- **Compliance**: OWASP API1:2023 BOLA — Critical severity

## Reproduction Steps
1. User in School A creates timetable slot (gets ID `slot-uuid-1`)
2. User in School B calls `DELETE /api/timetable/class-timetable-slots/slot-uuid-1`
3. Observe 204 No Content — slot deleted from School A

## Proposed Solutions
- **Option A (Fix handler + service)**: 
  - Handler: extract `schoolID` from `c.Locals("active_school_id")`
  - Service: `DeleteClassTimetableSlot(ctx, schoolID, slotID)` with ownership check
- **Option B (RLS)**: Rely on PostgreSQL Row-Level Security policy on `class_timetable_slots` table
- **Option C (Both)**: Defense in depth — handler check + RLS policy

## Decision
Pending — recommend Option A + Option C (handler validation + RLS).

## Acceptance Criteria
- [ ] Handler extracts `active_school_id` from `c.Locals()`
- [ ] Service method signature: `DeleteClassTimetableSlot(ctx, schoolID, slotID)`
- [ ] Service verifies slot belongs to school before delete (returns 404 if not)
- [ ] Integration test: User A cannot delete User B's slot → 404
- [ ] RLS policy verified on `class_timetable_slots` table

## Links
- Security audit report: `security-report-extended.md`
- OWASP API1:2023 reference: `docs/reference/owasp-api-top10-2023.md#api12023---broken-object-level-authorization-bola`
- Handler: `backend/internal/api/timetable_handler.go:168`
- Service: `backend/internal/services/timetable_service.go` (DeleteClassTimetableSlot)