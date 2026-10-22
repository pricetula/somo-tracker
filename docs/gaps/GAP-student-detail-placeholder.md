# Gap: Student Detail Page is Placeholder

**Area:** frontend
**Severity:** High
**Status:** Closed
**Owner:** Platform team
**Created:** 2026-10-15
**Resolved:** 2026-10-15
**Related Issues:** #

## Description
Student detail route `/students/[id]` renders only a placeholder with ID and no data.

## Current Behavior
`frontend/src/features/students/components/student-detail.tsx` returns:
```tsx
<h1>Student</h1>
<p>Student ID: {id}</p>
<p>Name placeholder</p>
```

## Expected Behavior
Full profile page with bio, class, guardians, attendance summary, edit capability.

## Impact
Admins cannot view or edit student details in UI. Forces use of lists only.

## Reproduction Steps
1. Go to Students list
2. Click any student name
3. See placeholder

## Proposed Solutions
- Implement Enhanced Student Profile feature spec
- Reuse existing ListStudents API + wire existing `GET /students/:id` and `PATCH /students/:id` handlers currently defined in `backend/internal/api/students_handler.go` but not registered in router
- Frontend fetch with TanStack Query, replace placeholder component

## Current Findings
- Backend handlers `GetStudent` and `UpdateStudent` exist in code but not routed in `backend/internal/api/router.go`
- Frontend component has scaffolded tabs and TODO comments for data fetching

## Acceptance Criteria
- [x] Page loads student data
- [x] Shows core fields from schema
- [x] Navigation works without dumb back button

## Links
- Feature: `docs/features/FEATURE-enhanced-student-profile.md`
