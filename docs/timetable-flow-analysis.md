# Timetable API/Frontend Flow Analysis & Remediation

**Date**: 2026-08-27  
**Status**: Draft — revised after UX discussion  
**Scope**: `backend/internal/timetable/` + `frontend/src/features/timetable/`

---

## Executive Summary

The timetable feature currently collapses two distinct responsibilities into a single page and a single wizard:
- **List & manage tracks** (a school has 0, 1, or N tracks)
- **Edit blocks and assign teachers** (per track)

The fix is **not** a database change. It's a **page-level decomposition** of the UI and a corresponding API surface cleanup. This document defines the new flow and the gaps in the current implementation against it.

---

## Revised Architecture: 3-Page Flow

```
/timetable                          → Track list
/timetable/new                      → Create track (details only)
/timetable/[trackId]                → Track detail (blocks grid + assignments)
/timetable/[trackId]/edit           → Edit track (name, description)
/timetable/[trackId]/blocks/new     → Add time block
/timetable/[trackId]/blocks/[blockId]/edit → Edit time block
/timetable/[trackId]/allocate       → Assign teacher (class + subject + teacher + room)
```

### Why this works
1. **No migration required.** `cbc_classes` stays untouched. The class context moves from "track owns classes" to "allocation references a class". One block can host N allocations (one per class) — same as today.
2. **Each page has a single job.** A user opening the timetable module sees the list first, then drills in. No more "the grid is empty because no blocks, or no tracks, or both?" confusion.
3. **Empty states are obvious.** The track list page handles 0-tracks. The track detail page handles 0-blocks. No shared fallback that masks both.

---

## Page-by-Page Flow

### Page 1: `/timetable` — Track List

**Purpose**: Show all timetable tracks for the active school + academic year.

**UI**:
- Page title: "Timetables"
- Top-right: "New Timetable" button → `/timetable/new`
- Body: cards or table of tracks, each showing:
  - Track name
  - Description (truncated)
  - Block count (e.g., "21 blocks across 7 days")
  - Allocation count (e.g., "84 assignments")
  - "Open" button → `/timetable/[trackId]`

**Edge cases**:
| Scenario | UI Behavior |
|----------|-------------|
| **0 tracks** | Empty state: illustration + "No timetables yet" + "Create your first timetable" CTA → `/timetable/new` |
| **1 track** | Render single card as normal |
| **3+ tracks** | Render all cards in a grid (responsive: 1/2/3 columns) |

**Backend requirement** (new):
```
GET /api/v1/timetable/tracks
  → { items: TimetableTrack[], total: number }
  Each track enriched with: block_count, allocation_count
```

**Frontend requirement** (new):
- `useTracks()` hook calling the new endpoint
- `TracksListPage` component
- Replace the current `timetable/page.tsx` content (currently shows the grid)

---

### Page 2: `/timetable/[trackId]` — Track Detail (Blocks Grid)

**Purpose**: Show the time blocks for one track and the assignments within them.

**UI**:
- Header bar:
  - Back arrow → `/timetable`
  - Track name (h1)
  - Description (subtitle)
  - Actions menu: "Edit Track" / "Delete Track"
- "Add Time Block" button (top-right of grid) → `/timetable/[trackId]/blocks/new`
- **Grid view** (existing component, refactored):
  - Rows: time blocks, ordered by `start_time`
  - Columns: days (1–7, but default 1–5 for the school week)
  - Cells:
    - Empty (no allocation for any class) → "Assign" link → `/timetable/[trackId]/allocate?block=<id>&day=<n>`
    - Has allocations → list of cards (one per class):
      - Subject (learning area)
      - Teacher
      - Class name
      - Room
      - Trash icon to remove
  - Break blocks: rendered as a styled row spanning all days, no "Assign" link, not clickable

**Edge cases**:
| Scenario | UI Behavior |
|----------|-------------|
| **Track exists, 0 blocks** | Grid placeholder + "No time blocks yet" + prominent "Add Time Block" CTA |
| **Track exists, blocks but 0 allocations** | Grid shows all time blocks, every cell is an "Assign" link |
| **Track exists, blocks with allocations** | Normal grid with rendered assignments |
| **Block has multiple allocations (multiple classes)** | Cell shows stacked assignment cards |

**Backend requirements**:
1. `GET /api/v1/timetable/tracks/[id]` → returns the track (or extend `GET /api/v1/timetable` to include `tracks[]`)
2. The current `GET /api/v1/timetable` is already correct (returns blocks + allocations); just need the track list to drive the grid scoping
3. `TimeBlock` response must include `track_id` so the frontend can filter (already present in DB)

**Frontend requirements**:
- New `useTrack(trackId)` hook
- Refactor `timetableViewSelect()` to accept a `trackId` filter
- Replace `AllocationBlock` cell-empty state to link to the new allocate page
- Add a "Day 6/7" toggle (most schools are Mon–Fri)

---

### Page 3: `/timetable/[trackId]/allocate` — Assignment Form

**Purpose**: Assign a (class, subject, teacher) to a specific (block, day) cell.

**Query params** (from the grid cell):
```
?block=<blockId>&day=<dayOfWeek>
```

**UI**:
- Page title: "Assign Teacher"
- Context line: "For [Track Name] · [Period Name] · [Day Name] · [Time Range]"
- Form fields:
  - Class (select) — required
  - Learning Area (select) — required
  - Teacher (select) — required
  - Room (optional text)
- "Save" button → `POST /api/v1/timetable/allocations`
- "Cancel" → back to track detail

**Edge cases**:
| Scenario | UI Behavior |
|----------|-------------|
| **Block is a break** (`is_break = true`) | Should never reach here (grid hides the "Assign" link for breaks) |
| **Class already has an allocation for this block** | 409 conflict → show error "This class is already assigned for this period" |
| **Teacher is double-booked** | 409 conflict → "This teacher is already teaching during this period" |
| **Room is double-booked** | 409 conflict → "This room is in use during this period" |
| **Block doesn't exist** (e.g., deleted) | 404 → "This time block no longer exists. Return to track" |

**Backend requirements**:
- Existing `POST /api/v1/timetable/allocations` works
- Add server-side validation: reject if block `is_break = true` (currently allowed — gap to fix)

---

### Page 4: `/timetable/new` — Create Track

**Purpose**: Create a new timetable track (name + description only, no blocks).

**UI** (simplified from current 2-step wizard):
- Form: name (required), description (optional)
- "Create" button → `POST /api/v1/timetable` (with empty `initial_blocks` or removed field)
- After success: navigate to `/timetable/[newTrackId]`

**Rationale for removing block creation from this page**:
- Today's wizard does details → blocks, but blocks are optional and most users skip them initially
- The track detail page now handles "this track has no blocks, add them" naturally
- Simpler first step; users get to a useful page (the empty track detail) faster

**Backend requirement**:
- `CreateTrackPayload` can keep `initial_blocks` as optional (for power users who want a one-shot track+blocks creation)
- Or split into two endpoints: `POST /api/v1/timetable/tracks` and `POST /api/v1/timetable/tracks/[id]/blocks` (cleaner REST)

**Frontend requirement**:
- Strip block creation from `CreateTimetable` component
- Make it a single-page form
- After success, redirect to `/timetable/[newTrackId]`

---

### Page 5: `/timetable/[trackId]/blocks/new` — Add Time Block

**Purpose**: Add one or more time blocks to a track.

**UI**:
- Page title: "Add Time Block"
- Context: "For [Track Name]"
- Form (replicates current `TimetableCreateBlocks` UI):
  - Period name (e.g., "Lesson 1", "Morning Break")
  - Start time / End time
  - "Is break" checkbox
  - "+ Add Another" button (for bulk add)
- "Save" button → `POST /api/v1/timetable/blocks` with all new blocks
- After success: navigate back to `/timetable/[trackId]`
- **Important**: backend currently replicates each block to all 7 days. UI should make this explicit: "This will apply to all 7 days of the week" with a checkbox to scope to selected days.

**Edge cases**:
| Scenario | UI Behavior |
|----------|-------------|
| **No tracks exist (direct URL access)** | Redirect to `/timetable` (list) |
| **Block time conflicts with existing** | 409 → "This period overlaps with [Existing Period]" |
| **Track is deleted mid-flow** | 404 → "This track no longer exists" |

**Backend requirement**:
- Existing `POST /api/v1/timetable/blocks` works
- The 7-day replication should be opt-in via a flag in the payload (e.g., `apply_to_all_days: bool`)

---

## State Diagrams

### Track Lifecycle (revisited)

```
[No Tracks]              [Has Tracks]              [Track Detail Page]
    │                          │                          │
    │                          │                          │
    ▼                          ▼                          ▼
Empty state           Track list (cards)         Blocks grid view
"Create your                                        │
 first timetable"                                    ├── 0 blocks: "Add Time Block" CTA
    │                                                ├── 0 allocations: "Assign" links
    ▼                                                └── Has allocations: rendered cells
/timetable/new                                               │
    │                                                   │
    └─── Create Track ──────────────────────────────→ Track Detail (empty blocks)
```

### Allocation Lifecycle (Per Block, Per Class)

```
[Cell in grid — no allocation]
    │
    ▼
Click "Assign"
    │
    ▼
/timetable/[trackId]/allocate?block=<id>&day=<n>
    │
    ├── Fill form (class, subject, teacher, room)
    │       │
    │       ├── Submit → 201 Created
    │       │           │
    │       │           ▼
    │       │       Redirect to track detail — cell now shows the allocation
    │       │
    │       └── Submit → 409 Conflict
    │                   │
    │                   ▼
    │               Error message ("already assigned" / "double-booked")
    │
    └── Cancel → back to track detail
```

### Block Lifecycle

```
[Track Detail — 0 blocks]
    │
    ▼
"Add Time Block"
    │
    ▼
/timetable/[trackId]/blocks/new
    │
    ├── Define period + times
    │       │
    │       ├── Save (1 block)
    │       │       ├── Backend: optionally replicate to all 7 days
    │       │       │
    │       │       └── Return to track detail (grid now shows blocks)
    │       │
    │       └── Save (N blocks — bulk)
    │               │
    │               └── Return to track detail
    │
    └── Back → track detail (no change)
```

---

## Open Questions

### Q1: Does the user need to pick a class before seeing the grid?

Currently an allocation is `(block, class, teacher, subject)`. The grid shows all allocations for a block (across all classes). Two options:

- **Option A (current model)**: Grid shows all classes stacked in each cell. Class context only appears at the allocation form. Simple grid, picks class at allocation time.
- **Option B**: A "class selector" dropdown above the grid. Grid filters to show only that class's allocations. Other classes' allocations are hidden but still exist.

**Recommendation**: Option A (current model) — less complexity, no migration needed. Class is picked at the allocation form.

### Q2: Should `initial_blocks` be removed from `CreateTrackPayload`?

Yes. Block creation should be a separate step on the track detail page. `POST /api/v1/timetable` becomes a pure track creation endpoint. Keep `initial_blocks` as deprecated/ignored or remove it entirely.

### Q3: Who can see which tracks?

Currently no role-based visibility on tracks. Every authenticated user sees all tracks for their school. Keep this simple for now — scope is `tenant_id + school_id`.

---

## Implementation Plan

### Step 1: Backend — Add tracks list endpoint
- `GET /api/v1/timetable/tracks` — returns tracks with `block_count`, `allocation_count`
- No schema change needed
- Estimated: 1 endpoint + 1 repo method + 1 service method

### Step 2: Frontend — Replace `/timetable` with track list page
- New `useTracks()` hook
- New `TracksListPage` component replacing current `timetable/page.tsx` content
- Empty state: "No timetables" + "Create Timetable" CTA
- Card grid: name, description, block/alloc counts, "Open" button

### Step 3: Backend — Add `GET /api/v1/timetable/tracks/[id]`
- Returns single track with blocks + allocations joined
- Or reuse existing `GET /api/v1/timetable` but filter by `track_id` (add `?track_id=` param)

### Step 4: Frontend — Track detail page `/timetable/[trackId]`
- Reuse existing `TimeTable` grid component, but:
  - Add track header (name, description, back button)
  - Add "Add Time Block" button
  - Update cell links to new allocate route
  - Skip "Assign" link for `is_break = true` cells
- New `useTrackBlocks(trackId)` hook

### Step 5: Backend — Reject allocations on break blocks
- Add validation in `CreateAllocation` service: check `block.is_break` → return 409

### Step 6: Frontend — Simplify `/timetable/new`
- Strip block creation step from wizard
- Single form (name + description) → `POST /api/v1/timetable`
- On success: navigate to `/timetable/[newTrackId]`

### Step 7: Frontend — New allocate page `/timetable/[trackId]/allocate`
- Read `?block=` and `?day=` from query params
- Class selector + learning area + teacher + room form
- `POST /api/v1/timetable/allocations`
- On success: navigate back to track detail

### Step 8: Frontend — New add blocks page `/timetable/[trackId]/blocks/new`
- Reuse/refactor `TimetableCreateBlocks` component
- Add `apply_to_all_days` toggle
- `POST /api/v1/timetable/blocks`
- On success: navigate back to track detail

---

## What Was Removed From the Original Document

- ❌ `cbc_classes.track_id` migration (not needed — class context lives in allocation)
- ❌ Multi-track tab logic (tracks are listed on `/timetable`, not merged into one grid)
- ❌ Academic year scoping (separate concern, deferred to a later phase)
- ❌ `TimetableRow.allocationByDay` → `Record<classId, Allocation>` (keep flat for now, stack cards in cell)
- ❌ `GET /api/v1/timetable/allocations` endpoint (not needed — grid uses combined view)

---

*End of document.*

---

## Parallel Routes (Interception) — Confirmed Pattern

All new action pages should be intercepted so they open as dialogs over the track detail grid, not as full page navigations.

### Existing pattern (already in repo)
```
/app/(dashboard)/@modal/(.)timetable/new/page.tsx     → modal overlay for /timetable/new
/app/(dashboard)/@modal/(.)timetable/allocate/page.tsx → modal overlay for /timetable/allocate
/app/(dashboard)/@modal/default.tsx                  → fallback, renders null
```

### New parallel routes for the 3-page flow

**Track list (`/timetable`)**
- Full page: `app/(dashboard)/timetable/page.tsx` (replaces current grid)
- No interception needed (this is the root entry point)

**Track detail (`/timetable/[trackId]`)**
- Full page: `app/(dashboard)/timetable/[trackId]/page.tsx`
- This is the base page that must remain visible under any modal

**Add Time Block — intercepted**
- Direct URL (full page): `app/(dashboard)/timetable/[trackId]/blocks/new/page.tsx`
- Modal overlay: `app/(dashboard)/@modal/(.)timetable/[trackId]/blocks/new/page.tsx`
- When navigating from track detail, the `(.)` route intercepts → dialog opens
- Direct navigation to full URL renders standalone form page (good for deep links)

**Assign Teacher — intercepted**
- Direct URL (full page): `app/(dashboard)/timetable/[trackId]/allocate/page.tsx`
- Modal overlay: `app/(dashboard)/@modal/(.)timetable/[trackId]/allocate/page.tsx`

**Edit Track — intercepted (optional)**
- Direct URL: `app/(dashboard)/timetable/[trackId]/edit/page.tsx`
- Modal overlay: `app/(dashboard)/@modal/(.)timetable/[trackId]/edit/page.tsx`

### How the modal pages work
Each intercepted page returns a `Dialog` (same as current `timetable/new` modal):

```tsx
export default function BlockCreateModal() {
  const router = useRouter();
  const handleRouteBack = () => router.back();
  return (
    <Dialog open onOpenChange={handleRouteBack}>
      <DialogContent>
        <DialogHeader>...</DialogHeader>
        <BlockCreateForm handleRouteBack={handleRouteBack} />
      </DialogContent>
    </Dialog>
  );
}
```

The grid cell's "Add Time Block" button links to `/timetable/[trackId]/blocks/new`. Because the link is clicked from within `/timetable/[trackId]`, the `(.)timetable/[trackId]/blocks/new` route matches and renders the modal overlay. The URL updates, the track detail stays visible underneath, and the user can close with `router.back()`.

### Direct navigation vs. interception
| Navigation type | What renders |
|-----------------|--------------|
| Click "Add Block" from `/timetable/[trackId]` | Modal overlay over grid |
| Direct `/timetable/[trackId]/blocks/new` | Full standalone page |
| Refresh on `/timetable/[trackId]/blocks/new` | Full standalone page (interception only applies when navigating from parent) |

---

## Confirmed Edge-Case Flow (with parallel routes)

### 0 tracks — `/timetable`
- Full page: empty state + "Create Timetable" CTA → `/timetable/new`
- If clicked: `/timetable/new` opens as modal over `/timetable`, or full page if direct

### 1 track — `/timetable` then `/timetable/[trackId]`
- `/timetable`: card shown → click → `/timetable/[trackId]`
- `/timetable/[trackId]`: grid shown; click cell 
  - No allocation: links to `/timetable/[trackId]/allocate` → modal opens
  - Blocks missing: "Add Time Block" → `/timetable/[trackId]/blocks/new` → modal opens

### 3 tracks — `/timetable`
- 3 cards shown → pick one → same as above
- Each track has its own detail path (`[trackId]`), no merged grid

### Track exists, 0 blocks — `/timetable/[trackId]`
- Grid empty
- "Add Time Block" button visible (top-right, always)
- Clicking opens `/timetable/[trackId]/blocks/new` modal
- After save, grid updates (re-fetch via React Query invalidation)

### Block exists, 0 allocations — `/timetable/[trackId]`
- Grid shows time blocks; all cells show "Assign" link
- Click → `/timetable/[trackId]/allocate` modal
- After save: grid updates (allocation appears in cell)

### Multiple allocations in one cell (2 classes same block)
- Cell renders stacked cards (one per class)
- Each card has its own delete button
- No need for another modal — just the grid cell

---

## Backend Handler Updates Required

Reference: `backend/internal/timetable/handler.go`

### 1. NEW: `GET /api/v1/timetable/tracks`
- **Purpose**: `/timetable` page (track list with counts)
- **Handler**: `ListTracks(c *fiber.Ctx)`
- **Service**: `ListTracks(ctx, tenantID, schoolID, yearID)`
- **Repo**: query `timetable_tracks` + aggregate counts from `timetable_blocks` / `timetable_allocations`
- **Response**: `{ items: Track[], total: int }` where each item can include `block_count` / `allocation_count`

### 2. UPDATE: `GET /api/v1/timetable` (`GetTimetable`)
- **Purpose**: `/timetable/[trackId]` grid needs blocks + allocations scoped to one track
- **Change**: accept optional `?track_id=` query param
- **Service**: pass `trackID` through `AllocationFilter` / `ListBlocks`
- **Repo**: filter `timetable_blocks` and `timetable_allocations` by `track_id` (via block → track join for allocations)
- **Also**: include `track_id` in `TimeBlock` response so frontend can group/verify

### 3. UPDATE: `POST /api/v1/timetable/blocks` (`CreateBlocks`)
- **Purpose**: `/timetable/[trackId]/blocks/new` — add blocks to existing track
- **Change**: already supports this; just add validation that `track_id` belongs to the school/tenant
- **Optional improvement**: if payload includes `apply_to_all_days`, replicate to 7 days; if not, create only the specified `day_of_week` (current behavior replicates always, which is fine if UI makes it explicit)

### 4. UPDATE: `POST /api/v1/timetable/allocations` (`CreateAllocations`)
- **Purpose**: `/timetable/[trackId]/allocate` — assign teacher to cell
- **Critical gap**: add validation — reject if `block.is_break == true`
- **Implementation**: in service layer, fetch block by `block_id`; if `is_break`, return `ErrConflict` with message "cannot assign to a break block"
- **Also**: current `allocation-block.tsx` skips the link for breaks, but server-side guard prevents direct API calls

### 5. UPDATE: `POST /api/v1/timetable` (`CreateTrackWithBlocks`)
- **Purpose**: `/timetable/new` stripped to details-only
- **Change**: `initial_blocks` can be deprecated/ignored, or kept optional. The wizard no longer sends it, but backend can still handle it for backward compatibility.
- **No schema change needed.**

### 6. NO CHANGE (already correct):
- `PUT /api/v1/timetable` — edit track (`/timetable/[trackId]/edit`)
- `DELETE /api/v1/timetable` — bulk delete tracks (from list page actions)
- `PUT /api/v1/timetable/blocks` — edit block
- `DELETE /api/v1/timetable/blocks` — bulk delete blocks
- `PUT /api/v1/timetable/allocations` — edit assignment
- `DELETE /api/v1/timetable/allocations` — delete assignment

---

## Route Registration Updates

In `RegisterRoutes`, add the new list endpoint:

```go
base.Get("/tracks", middleware.RequireAuth, h.ListTracks)
// Optionally: base.Get("/tracks/:id", middleware.RequireAuth, h.GetTrack)
```

The `GetTimetable` route stays at `base.Get("/", ...)` but gains query param support.

---

## Summary — What Changes, What Doesn't

| Handler | Status | Action |
|---------|--------|--------|
| `CreateTrackWithBlocks` | Minor | Allow empty `initial_blocks`; wizard stops sending them |
| `ListTracks` | **NEW** | List page endpoint |
| `GetTimetable` | Update | Add `?track_id=` filter |
| `CreateBlocks` | No change | Already supports track-scoped creation |
| `CreateAllocations` | **Update** | Reject allocations on `is_break = true` |
| `UpdateTrack` / `DeleteTrack` / etc. | No change | Used by detail/edit actions |

---

---

## Block Updates/Deletes by Period Name (Not Block ID)

Confirmed: block modifications should target the logical period, not individual rows, because each `period_name` is replicated to all 7 days.

### Current model (per-block)
```
UpdateBlock  →  payload has `id`  →  updates 1 row (1 day)
BulkDelete  →  payload has `ids[]`  →  deletes N rows (N individual days)
```

### Proposed model (per-period, within a track)
The identifier for a time block should be `(track_id, period_name)` — because replication creates 7 rows (Mon–Sun) with identical `period_name`.

```
Update period "Lesson 1" in track X:
  → find all blocks where track_id = X AND period_name = "Lesson 1"
  → update start_time, end_time, is_break on all 7 rows

Delete period "Morning Break" in track X:
  → find all blocks where track_id = X AND period_name = "Morning Break"
  → delete all 7 rows (and cascade to allocations)
```

### Handler changes needed

**Update (`PUT /api/v1/timetable/blocks`)**
- Option A (modify): change payload from `id` to `track_id` + `period_name`, find all matching rows
- Option B (new endpoint): `PUT /api/v1/timetable/blocks/period` with `{ track_id, period_name, start_time?, end_time?, is_break? }`
- **Recommendation**: Option B — avoids breaking existing per-block edit (if needed for edge cases like changing one day)

**Delete (`DELETE /api/v1/timetable/blocks`)**
- Change payload from `{ ids: string[] }` to `{ track_id, period_name }` (or `{ track_id, period_names: string[] }` for bulk period delete)
- **Service**: `DeleteBlockByPeriod(ctx, trackID, periodName)` → deletes all `timetable_blocks` for `(track_id, period_name)` cascade to allocations

**Why this fits the replication model**
When `CreateBlocks` replicates to all 7 days, the user thinks of "Lesson 1" as one entity, not 7 separate DB rows. Updating by `period_name` respects that mental model.

---
---

## Note Added: Dedicated Track Detail Endpoint
`GET /api/v1/timetable` returns all blocks + allocations (combined view). For `/timetable/[trackId]` the frontend should call `GET /api/v1/timetable/tracks/[id]` (not the combined endpoint) to retrieve the specific track with its block/allocation context.
