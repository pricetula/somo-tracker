# Gap: Bulk Import UI

**Area:** frontend
**Severity:** High
**Status:** Open
**Owner:** 
**Created:** 2026-10-15
**Related Issues:** #

## Description
Backend bulk operations framework is implemented but there is no frontend UI for admins to upload files, monitor job progress, or retry failed rows.

## Current Behavior
Bulk jobs can only be created via API / scripts. No visibility in the dashboard.

## Expected Behavior
Admins can upload CSV/Excel, map columns, preview, submit, and see real-time progress with per-row errors and retry.

## Impact
Operators cannot use bulk import in production. Manual entry is slow and error-prone.

## Reproduction Steps
1. Log in as school admin
2. Navigate to Students / Staff management
3. No bulk upload button visible

## Proposed Solutions
- Option A: Build a dedicated Bulk Import page with drag-and-drop, similar to shadcn file-upload component
- Option B: Add bulk actions to existing tables with CSV import modal

## Decision
TBD

## Acceptance Criteria
- [ ] Upload UI for student import with validation
- [ ] Job status polling / websocket updates
- [ ] Error table with retry per row
- [ ] Idempotency key shown to user

## Links
- Feature: `docs/features/FEATURE-bulk-jobs.md`
