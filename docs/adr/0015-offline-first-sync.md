# ADR-0015: Offline-First Data Capture & Sync

## Metadata

| Field | Value |
|-------|-------|
| id | ADR-0015 |
| title | Offline-First Data Capture & Sync |
| status | proposed |
| date | 2026-11-25 |
| author | frontend, backend |
| deciders | frontend, backend |
| relates_to | ADR-0010, ADR-0006, SPEC-002 |

## Status

proposed

## Context

Schools in East Africa frequently experience intermittent or limited internet connectivity. ADR-0010 claims assessments work "offline or in environments with limited internet access," but no mechanism exists for capturing data offline and synchronizing when connectivity returns.

The frontend is a Next.js BFF proxy (ADR-0006); all data flows through Next.js to the Go backend. An offline strategy must account for this architecture.

## Decision

Adopt an **optimistic local-first capture with background sync** model:

1. **Local write queue**: When the browser is offline, write mutations (attendance, assessment results) to IndexedDB/localStorage with a `synced` flag and timestamp. Reads show locally cached data optimistically.

2. **Sync on reconnect**: A background process detects online status and flushes the local write queue to the backend via the BFF proxy (`/api/auth/*` and future API routes). Writes are sent in chronological order.

3. **Conflict resolution**: Last-write-wins based on `updated_at` timestamp. If a record was modified on the server while offline, the server version wins and the local change is discarded with a UI notification to the teacher.

4. **No offline authentication**: OTP login and session validation require connectivity (email delivery + backend validation). Offline mode applies only to data capture (attendance, assessments) for already-authenticated users with a valid session cookie.

5. **API design for sync**: Future API endpoints must support partial batch writes and return per-item success/failure so the sync client can retry individual failures.

## Alternatives Considered

| Alternative | Pros | Cons | Why rejected |
|-------------|------|------|--------------|
| Service Worker + background sync | Native browser offline support | Complex, limited browser support, debugging difficulty | Rejected — IndexedDB simpler for MVP |
| Read-only offline | Simple | Cannot capture data without internet | Rejected — defeats offline purpose |
| Local-first queue (chosen) | Captures data offline, syncs when online | Conflict handling needed | Accepted |

## Consequences

### Positive

- Teachers can record attendance and assessments without internet.
- Data is not lost during connectivity outages.
- Matches the "works offline" claim in ADR-0010.

### Negative

- Conflict resolution may discard local changes (teacher must re-enter).
- Sync logic adds frontend complexity.
- Authentication still requires connectivity.

### Neutral / Risks

- IndexedDB storage limits are generous but not unlimited; large assessment batches may need chunking.
- Session cookie expiry during prolonged offline use will block sync until re-authentication.

## Links

- BFF proxy: `docs/adr/0006-bff-proxy-architecture.md`
- Local assessments: `docs/adr/0010-local-assessments-only.md`
- Attendance: `docs/adr/0012-attendance-tracking.md`
