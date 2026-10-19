# Gap: Timetable Handlers Use Inline Request DTOs Without Validation

**Area:** backend
**Severity:** Low
**Status:** Open
**Owner:** 
**Created:** 2025-10-19
**Related Issues:** #security-audit-2025-10-19

## Description
Timetable handlers use inline struct definitions for request payloads without validation tags, reducing maintainability and allowing invalid data through.

## Current Behavior
**CreateTemplate** (`timetable_handler.go:26-34`):
```go
type createTimetableTemplateRequest struct {
    Name        string `json:"name" validate:"required"`
    Description string `json:"description"`
    TimeSlots   []struct {  // Inline struct — no validation tags
        Name            string `json:"name"`
        StartTime       string `json:"start_time"`
        EndTime         string `json:"end_time"`
        IsInstructional bool   `json:"is_instructional"`
    } `json:"time_slots"`
}
```

**SetupClassTimetableSlot** (`timetable_handler.go:203-212`):
```go
var req struct {
    ClassRoomID         string `json:"class_room_id"`
    DayOfWeek           int    `json:"day_of_week"`
    TimeSlotID          string `json:"time_slot_id"`
    SubjectID           string `json:"subject_id"`
    TeacherMembershipID string `json:"teacher_membership_id"`
    RoomID              string `json:"room_id"`
}
// No validate tags on UUID fields
```

## Expected Behavior
- Extract inline structs to named types in `timetable_handler.go` or shared `dtos.go`
- Add validation tags: `validate:"required,uuid"` for ID fields, `validate:"required,min=1,max=7"` for DayOfWeek
- Use `time.Time` or validated string format for `StartTime`/`EndTime`

## Impact
- **Maintainability**: Inline structs hard to reuse, test, document
- **Validation Gaps**: No format validation on UUIDs, time strings
- **Mass Assignment Risk**: If fields added to inline struct, automatically accepted
- **OpenAPI**: Inline structs may not generate proper schema

## Reproduction Steps
1. POST `/api/timetable/setup` with `class_room_id: "not-a-uuid"` → accepted, fails later in service
2. POST with `day_of_week: 99` → accepted, fails later

## Proposed Solutions
- **Option A (Named DTOs)**: Extract to named structs with validation tags
- **Option B (Shared package)**: Create `backend/internal/api/dtos/timetable.go` for reuse

## Decision
Pending — recommend Option A for quick win.

## Acceptance Criteria
- [ ] `createTimetableTemplateRequest.TimeSlots` extracted to named `CreateTimeSlotRequest` struct
- [ ] `SetupClassTimetableSlot` request extracted to named struct
- [ ] All ID fields have `validate:"required,uuid"`
- [ ] `DayOfWeek` has `validate:"required,min=1,max=7"`
- [ ] `StartTime`/`EndTime` have time format validation
- [ ] Unit tests for validation

## Links
- Security audit report: `security-report-extended.md`
- OWASP API3:2023 reference: `docs/reference/owasp-api-top10-2023.md#api32023---broken-object-property-level-authorization-bpla`
- Handler: `backend/internal/api/timetable_handler.go:26, 203`