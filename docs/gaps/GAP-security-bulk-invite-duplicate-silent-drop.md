# Gap: Bulk Invitation Duplicate Emails Silently Dropped

**Area:** backend
**Severity:** High
**Status:** Open
**Owner:** 
**Created:** 2025-10-19
**Related Issues:** #security-audit-2025-10-19

## Description
Bulk invitation endpoints deduplicate emails case-insensitively but silently drop duplicates without informing the caller. User submits 100 rows with 50 duplicates → only 50 invitations sent, no feedback.

## Current Behavior
In `admin_invitation_handler.go:115-118` and `teacher_invitation_handler.go:115-118`:
```go
lower := strings.ToLower(email)
if _, exists := seen[lower]; exists {
    // Skip duplicate silently – first occurrence wins
    continue
}
seen[lower] = struct{}{}
dedupedRows = append(dedupedRows, InvitationRow{Email: email, FullName: fullName})
```

## Expected Behavior
Response should include count of deduplicated rows and optionally which indices were dropped.

## Impact
- **Silent Data Loss**: User thinks 100 invitations sent, only 50 processed
- **Debugging Difficulty**: No visibility into why fewer invitations than expected
- **User Experience**: Confusing — "I uploaded 100 but only 50 got invites"

## Reproduction Steps
1. POST `/api/admins/invitations` with 10 rows, 5 duplicate emails (case variations)
2. Observe response: `total_records: 5` (no mention of 5 dropped)
3. Check job items — only 5 created

## Proposed Solutions
- **Option A (Warning in response)**: Add `deduplicated_count` and `deduplicated_indices` to response
- **Option B (Fail on duplicate)**: Return 400 if duplicates detected (strict)
- **Option C (Client-side dedup)**: Document that clients should dedupe; server rejects duplicates

## Decision
Pending — recommend Option A (informative warning) for best UX.

## Acceptance Criteria
- [ ] Response includes `deduplicated_count: 5` when duplicates dropped
- [ ] Optionally includes `deduplicated_indices: [2, 5, 7, ...]` (0-based)
- [ ] Response message updated: "Bulk invitation job registered successfully (5 duplicates removed)"
- [ ] Integration test: duplicates in request → response includes dedup count
- [ ] Applied to all bulk invitation endpoints

## Links
- Security audit report: `security-report-extended.md`
- OWASP API6:2023 reference: `docs/reference/owasp-api-top10-2023.md#api62023---unrestricted-access-to-sensitive-business-flows`
- Admin handler: `backend/internal/api/admin_invitation_handler.go:115`
- Teacher handler: `backend/internal/api/teacher_invitation_handler.go:115`