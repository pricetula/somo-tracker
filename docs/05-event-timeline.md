# 05 — Event Timeline & Public Holidays

## Overview
Implement annual timeline view combining school events and public holidays. Based on design in `docs/event-timeline-ui-plan.md`.

---

## Backend Changes

### 1. Public Holidays CRUD (Missing)
**File:** `backend/db/queries/public_holidays.sql` (new)

```sql
-- name: CreatePublicHoliday :one
INSERT INTO public_holidays (country_id, name, month, day, description)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetPublicHoliday :one
SELECT * FROM public_holidays WHERE id = $1;

-- name: ListPublicHolidays :many
SELECT * FROM public_holidays
WHERE country_id = $1
  AND ($2::int IS NULL OR month = $2)
  AND ($3::int IS NULL OR month = $3)
ORDER BY month, day
LIMIT $4 OFFSET $5;

-- name: CountPublicHolidays :one
SELECT COUNT(*) FROM public_holidays WHERE country_id = $1;

-- name: UpdatePublicHoliday :one
UPDATE public_holidays
SET name = $2, month = $3, day = $4, description = $5, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeletePublicHoliday :exec
DELETE FROM public_holidays WHERE id = $1;
```

### 2. School Events CRUD (Partial — only GET exists)
**File:** `backend/db/queries/school_events.sql` (extend existing)

Add: `CreateSchoolEvent`, `UpdateSchoolEvent`, `DeleteSchoolEvent`

### 3. Timeline Query (Combined)
**File:** `backend/db/queries/timeline.sql` (new)

```sql
-- name: GetTimelineEvents :many
-- Returns combined school events + public holidays for date range
WITH school_events AS (
    SELECT 
        id, title, event_type, start_date, end_date, requires_attendance,
        'school_event' as source, NULL as country_id
    FROM school_events
    WHERE school_id = $1
      AND start_date <= $3 AND end_date >= $2
),
public_holidays AS (
    SELECT 
        id, name as title, 'PUBLIC_HOLIDAY' as event_type,
        make_date($4, month, day) as start_date,
        make_date($4, month, day) as end_date,
        FALSE as requires_attendance,
        'public_holiday' as source, country_id
    FROM public_holidays
    WHERE country_id = $5
      AND make_date($4, month, day) BETWEEN $2 AND $3
)
SELECT * FROM school_events
UNION ALL
SELECT * FROM public_holidays
ORDER BY start_date, event_type;
```

### 4. Services & Handlers
- `PublicHolidaysService` + `PublicHolidaysHandler`
- Extend `EventsService` + `EventsHandler` with create/update/delete
- `TimelineService` + `TimelineHandler` with `GET /api/timeline`

### 5. Router Endpoints
```
GET    /api/public-holidays
POST   /api/public-holidays
GET    /api/public-holidays/:id
PATCH  /api/public-holidays/:id
DELETE /api/public-holidays/:id

POST   /api/events              # (add)
PATCH  /api/events/:id          # (add)
DELETE /api/events/:id          # (add)

GET    /api/timeline            # combined view
  Query: date_from, date_to
```

---

## Frontend Changes

### 1. Feature Module
```
src/features/event-calendar/
├── components/
│   ├── timeline-view.tsx       # Annual horizontal bar timeline
│   ├── calendar-view.tsx       # Monthly grid (existing Calendar component)
│   ├── event-form.tsx          # Create/edit event dialog
│   ├── holiday-form.tsx        # Create/edit holiday dialog
│   ├── timeline-bar.tsx        # Gantt-style bar component
│   └── event-tooltip.tsx       # Hover details
├── hooks/
│   ├── use-events.ts           # (extend existing)
│   ├── use-public-holidays.ts  # (new)
│   └── use-timeline.ts         # (new)
├── services/
│   └── api.ts
├── types/
│   └── event.ts
└── index.ts
```

### 2. Components

#### TimelineView (Annual Bar)
```tsx
// Horizontal scrollable timeline
// Each event = colored bar spanning start_date to end_date
// School events: primary color (indigo)
// Public holidays: amber/red
// Click → detail tooltip
```

#### CalendarView (Monthly Grid)
```tsx
// Use existing Calendar component
// dayContent prop shows dots/badges for events/holidays
// Color-coded by source
```

### 3. Pages
```
src/app/(dashboard)/events/
├── page.tsx              # Timeline view (default)
├── calendar/page.tsx     # Monthly calendar view
├── add/page.tsx          # Create event
├── [id]/page.tsx         # Event detail
└── holidays/page.tsx     # Public holidays management
```

### 4. Navigation
Already in nav: `{ title: "School Events", url: "/events", icon: <CalendarRange /> }`

---

## Acceptance Criteria
- [ ] Annual timeline: horizontal scrollable bar view
- [ ] Monthly calendar: grid with event indicators
- [ ] School events: full CRUD (title, type, dates, attendance flag)
- [ ] Public holidays: full CRUD (name, month, day, country)
- [ ] Combined timeline query (school events + holidays)
- [ ] Visual differentiation: colors, shapes, labels
- [ ] Filter: show/hide school events, show/hide holidays
- [ ] Click event → tooltip with details
- [ ] RLS: tenant isolation

---

## Dependencies
- Requires: Countries table (for public holidays country_id)
- Related: Attendance (events with `requires_attendance`)

---

## Estimated Effort
- Backend: ~8 hours
- Frontend: ~12 hours (timeline visualization)
- **Total: ~20 hours**