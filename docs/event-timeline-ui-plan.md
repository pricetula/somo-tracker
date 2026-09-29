# UI/UX Planning: Annual Timeline — School Events & Public Holidays

> Planning phase — design discussion only. No code changes committed.

---

## 1. Goal

Plan how the annual timeline component will display (and optionally manage) **school events** and **public holidays** through queries that paint data onto a visual timeline view.

---

## 2. Data Context (What Exists Today)

### Backend Tables
| Source | Table | Key Fields | Query File |
|---|---|---|---|
| School-specific | `school_events` | `id`, `school_id`, `title`, `event_type`, `start_date`, `end_date`, `requires_attendance` | `academic_calendar.sql` |
| Country-level | `public_holidays` | `id`, `country_id`, `name`, `month`, `day` | `academic_calendar.sql` |

### API Endpoints
- `GET /api/events` — returns school events only.
- **No endpoint** for `public_holidays` management or retrieval.
- **No endpoint** for creating/updating/deleting school events via the API.

### Frontend Components
- `components/shared/calendar.tsx` — reusable `react-day-picker` wrapper with `dayContent` prop for painting extra content inside each day cell.
- `features/event-calendar/` — scaffolding only (`types/event.types.ts`, `components/EventCard.tsx`, `hooks/useEvents.ts`). The service (`eventApi`) and container/view components are missing or not fully wired.

---

## 3. Planning Questions & Design Decisions

### 3.1 Timeline Component — What Does "Paint" Mean?

We need to clarify the visual model:

**Option A: Day-cell overlay (current pattern)**
- The existing `Calendar` component accepts `dayContent?: (date: Date) => React.ReactNode`.
- Each day cell shows dots, badges, or small indicators for events/holidays falling on that date.
- Good for dense data; limited space per cell.

**Option B: Horizontal/vertical bar timeline (Gantt-style)**
- A separate timeline component shows events as colored bars spanning `start_date` → `end_date`.
- Better for multi-day events (exams, sports weeks) and public holidays.
- Requires a different rendering layer from the current `Calendar` component.

**Recommendation for discussion:**
- Keep `Calendar` (Option A) for the monthly/annual grid view with dots/badges.
- Add a new **annual timeline bar component** (Option B) for visualizing duration-based events side-by-side.
- Both components can consume the same query results but render differently.

---

### 3.2 Queries — What Needs to Be Fetched?

To paint both sources onto the timeline, we need separate or combined queries.

**Proposed Query Strategy:**

```
Query 1: School Events (by date range)
  Source: school_events table
  Filter: school_id = current_school, start_date BETWEEN ?, end_date BETWEEN ?
  Returns: id, title, event_type, start_date, end_date, requires_attendance

Query 2: Public Holidays (by country, by date range)
  Source: public_holidays table
  Filter: country_id = current_country, (month, day) within range
  Note: public_holidays uses month/day, not full date — requires conversion for range queries
  Returns: id, name, month, day, country_id
```

**UI Consumption Pattern:**
- The timeline component receives **two data sources**.
- It maps each source to a different visual style (e.g., blue bars for school events, red markers for public holidays).
- The component handles merging/overlapping rendering internally.

---

### 3.3 Visual Design — How to Differentiate?

To avoid confusion between school events and public holidays on the same timeline, propose:

| Aspect | School Events | Public Holidays |
|---|---|---|
| Color / Badge | Primary school color (e.g., indigo / blue) | Accent / warning (e.g., amber / red) |
| Shape on timeline | Rounded bar spanning `start` → `end` | Diamond or dot (single-day events typically) |
| Label style | Event title + `event_type` tag (e.g., EXAM, SPORTS) | Holiday name + country code |
| Interaction | Click → event detail / attendance tracking | Click → informational tooltip only |

---

### 3.4 Interaction Model — Read vs. Manage

We need to separate the **read-only timeline view** from the **management interface**.

**Read Timeline (Display Only):**
- Annual or monthly view.
- Filter toggles: Show/Hide School Events, Show/Hide Public Holidays.
- Hover/click for details.

**Manage Interface (Edit/Create):**
- Needs a separate mode or overlay.
- School events: add/edit title, event_type, dates, attendance flag.
- Public holidays: currently has **no CRUD API** — requires backend endpoint design if management is needed.

**Open Design Question:**
- Should public holidays be manageable by school admins (to override national holidays with school-specific closures), or should they remain read-only (pulled from country-level data)?

---

### 3.5 Component Structure — Proposed Architecture

Based on the current `event-calendar` scaffolding:

```
features/event-calendar/
  types/
    event.types.ts        (existing — good)
  services/
    event-api.ts          (missing — needs query definitions for school events + public holidays)
  hooks/
    useEvents.ts          (existing — fetch school events by month range)
    usePublicHolidays.ts  (new — fetch public holidays by country/range)
  components/
    EventCalendarContainer  (exported but missing — should orchestrate data sources)
    EventCalendarView     (exported but missing — should render the timeline)
    EventCreateDialog     (exported but missing — create/edit form)
    TimelineRow / TimelineBar  (new — bar-style rendering of events across dates)
```

---

### 3.6 Query Implementation Details

**For `school_events`:**
- The existing `academic_calendar.sql` already covers this.
- The frontend `useEvents` hook uses `startOfMonth` / `endOfMonth` to build date ranges.
- For an annual timeline, the range would expand to a full year.

**For `public_holidays`:**
- Because `public_holidays` is stored by `month` and `day` (not full date), range queries need to filter on these columns or expand the lookup logic in SQL.
- A query like `SELECT * FROM public_holidays WHERE country_id = ? AND month BETWEEN ? AND ?` is possible, but crossing year boundaries requires care.
- Alternative: fetch all holidays for the country and filter client-side by month/day within the timeline range.

---

## 4. Key Decisions Needed Before Implementation

1. **Timeline Style:** Monthly grid (`Calendar`) vs. horizontal bar timeline vs. both?
2. **Public Holiday Management:** Read-only (national data) or editable (school-level overrides)?
3. **Event Type Filtering:** Should `event_type` (SPORTS, EXAM, ADMISSION) be a filter dimension on the timeline?
4. **Data Merging:** Should school events and public holidays be fetched in a single combined endpoint, or kept as separate queries consumed by the same component?
5. **Mobile View:** How does the timeline render on narrow screens — vertical scroll vs. compressed view?

---

## 5. Implementation Roadmap (Phase Planning)

### Phase 1 — Query Foundation
- Design SQL query for public holidays by country/date range (`public_holidays`).
- Confirm `school_events` query supports full-year range (currently uses `start/end` range filter).
- Design API endpoint response format (combined or separate).

### Phase 2 — Component Design
- Design `TimelineRow` / `TimelineBar` component for bar-style rendering.
- Design visual styles (colors, badges, hover cards) for the two data sources.
- Plan the `EventCalendarContainer` data flow (how it combines `useEvents` + `usePublicHolidays`).

### Phase 3 — UI/UX Implementation
- Build read-only annual timeline view.
- Add filter controls (show/hide event types, show/hide public holidays).
- Build click/hover interactions (tooltips, detail panels).

### Phase 4 — Management Interface (Optional)
- Add create/edit dialogs for school events (`EventCreateDialog`).
- Add API endpoints for public holiday management (if decided in planning).

---

## 6. References

- `frontend/src/components/shared/calendar.tsx` — existing reusable calendar component.
- `frontend/src/features/event-calendar/` — event calendar scaffolding.
- `docs/tables-endpoints-queries.md` — database tables, queries, and endpoint inventory.
- `backend/db/migrations/000006_create_academic_calendar.up.sql` — schema definitions for `school_events` and `public_holidays`.
- `backend/internal/api/router.go` (implied) — endpoint routing.

---

*Document status: Discussion / Planning only. No code committed.*
