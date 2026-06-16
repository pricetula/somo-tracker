# Academic Calendar Validation Engine

This document describes the complete academic calendar setup and validation lifecycle — from the client-side evaluation decision tree through the Go backend API contract and PostgreSQL migration.

---

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Database Schema & Migration](#database-schema--migration)
- [Backend API Contract](#backend-api-contract)
  - [Go Module Structure](#go-module-structure)
  - [Endpoints](#endpoints)
- [Frontend TanStack Query Architecture](#frontend-tanstack-query-architecture)
- [Component State Decision Tree](#component-state-decision-tree)
- [Form Component (State A)](#form-component-state-a)
  - [is_final Toggle](#is_final-toggle)
  - [Anti-Clash Date Engine](#anti-clash-date-engine)
- [Mutation & Success Transitions](#mutation--success-transitions)
- [Dashboard Integration](#dashboard-integration)
- [File Reference](#file-reference)

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────┐
│  Next.js Frontend                                   │
│                                                     │
│  DashboardPage                                      │
│    └─ useCalendarEvaluator()  ← Decision Tree       │
│         ├─ "loading"   → Loader skeleton            │
│         ├─ "form"      → AcademicCalendarForm        │
│         ├─ "hidden"    → (dashboard unlocked)       │
│         └─ "hidden"    → PrepModeAlert              │
│              alert="prep-mode"                       │
│                                                     │
│  AcademicCalendarForm (State A)                     │
│    └─ useSaveAcademicCalendar() → POST API          │
│         └─ onSuccess → invalidateQueries()          │
│              → re-run decision tree → collapse form │
└──────────────────────┬──────────────────────────────┘
                       │
                       │ GET/POST /api/v1/schools/current-calendar
                       │ (session cookie: somo_sid)
                       ▼
┌─────────────────────────────────────────────────────┐
│  Go Backend (Fiber + Fx)                            │
│                                                     │
│  academiccalendar.Handler                           │
│    ├─ requireAuth middleware                        │
│    │    └─ auth.Service.GetSession()                │
│    ├─ GetCurrentCalendar (GET)                      │
│    └─ UpsertCurrentCalendar (POST)                  │
│         └─ transactional: unset → upsert → replace  │
│                                                     │
│  academiccalendar.Service                           │
│    └─ ResolveSchoolID() → membership → schools      │
│                                                     │
│  academiccalendar.Repository                        │
│    └─ pgxpool queries                               │
└──────────────────────┬──────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────┐
│  PostgreSQL                                         │
│                                                     │
│  academic_years: id, tenant_id, school_id, name,    │
│                  start_date, end_date, is_current    │
│                                                     │
│  academic_terms: id, tenant_id, academic_year_id,   │
│                  name, start_date, end_date,         │
│                  is_current, is_final ✨             │
└─────────────────────────────────────────────────────┘
```

---

## Database Schema & Migration

### Table: `academic_years`

| Column | Type | Notes |
|---|---|---|
| `id` | `UUID` | PK, `gen_random_uuid()` |
| `tenant_id` | `UUID` | FK → `tenants(id)` |
| `school_id` | `UUID` | FK → `schools(id)` |
| `name` | `VARCHAR(50)` | e.g. `"2026"` — maps to payload `year` |
| `start_date` | `DATE` | First period start |
| `end_date` | `DATE` | Last period end |
| `is_current` | `BOOLEAN` | Only one per school can be `true` (unique partial index) |

### Table: `academic_terms`

| Column | Type | Notes |
|---|---|---|
| `id` | `UUID` | PK |
| `tenant_id` | `UUID` | FK |
| `academic_year_id` | `UUID` | FK → `academic_years(id)` |
| `name` | `VARCHAR(100)` | e.g. `"Term 1"` |
| `start_date` | `DATE` | |
| `end_date` | `DATE` | Constraint: `end_date > start_date` |
| `is_current` | `BOOLEAN` | |
| **`is_final`** | **`BOOLEAN`** | **Added in migration 000003** — marks the final term of the academic year |

### Migration: `000003_add_is_final_to_terms`

```sql
DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'academic_terms' AND column_name = 'is_final'
    ) THEN
        ALTER TABLE academic_terms
            ADD COLUMN is_final BOOLEAN NOT NULL DEFAULT false;
    END IF;
END $$;
```

---

## Backend API Contract

### Go Module Structure

```
backend/internal/academiccalendar/
├── domain.go        — AcademicYear, AcademicPeriod, SavePayload types
├── repository.go    — pgx queries with transaction support
├── service.go       — business logic, school resolution
└── handler.go       — HTTP handlers, session auth middleware, fx.Module
```

### Endpoints

#### `GET /api/v1/schools/current-calendar`

Returns the current academic year with all its periods.

**Authentication:** Required (`somo_sid` cookie)

**Response `200 OK`:**
```json
{
  "id": "uuid-of-year",
  "year": 2026,
  "periods": [
    { "id": "uuid", "name": "Term 1", "start_date": "2026-01-06", "end_date": "2026-04-11", "is_final": false },
    { "id": "uuid", "name": "Term 2", "start_date": "2026-05-05", "end_date": "2026-08-15", "is_final": false },
    { "id": "uuid", "name": "Term 3", "start_date": "2026-09-01", "end_date": "2026-11-28", "is_final": true }
  ]
}
```

**Response `404 Not Found`:**
```json
{ "error": "not_found", "message": "no academic calendar configured yet" }
```

#### `POST /api/v1/schools/current-calendar`

Creates or replaces the current academic calendar. Runs **atomically** inside a database transaction:

1. Unsets `is_current = false` on all existing years for the school.
2. Upserts the year record (lookup by school + name; inserts if not found).
3. Deletes all existing terms for that year and inserts the new ones.
4. Commits.
5. Returns the fully hydrated calendar.

**Request:**
```json
{
  "year": 2026,
  "periods": [
    { "name": "Term 1", "start_date": "2026-01-06", "end_date": "2026-04-11", "is_final": false },
    { "name": "Term 2", "start_date": "2026-05-05", "end_date": "2026-08-15", "is_final": false },
    { "name": "Term 3", "start_date": "2026-09-01", "end_date": "2026-11-28", "is_final": true }
  ]
}
```

**Response `200 OK`:** Returns the saved calendar (same shape as GET).

**Validation:**
- `year` is required (must be non-zero).
- `periods` must have at least 1 entry.
- Individual period dates are not re-validated server-side (trust client-side validation).

### Session Auth Flow

The handler uses a `requireAuth` middleware that:

1. Reads the `somo_sid` cookie.
2. Calls `auth.Service.GetSession()` to validate and retrieve tenant + user IDs.
3. Stores `tenant_id` and `user_id` on the Fiber locals for downstream handlers.

School ID is resolved via membership lookup (user's active membership → school_id), with a fallback to the tenant's first active school.

---

## Frontend TanStack Query Architecture

### Global Cache Strategy

```typescript
// Query key, source, and config
const calendarKeys = { current: ["academic-calendar", "current"] as const };

useQuery({
  queryKey: calendarKeys.current,
  queryFn: fetchCurrentCalendar,
  staleTime: Infinity,          // Data changes exceptionally rarely
  refetchOnWindowFocus: false,  // No need to refetch on tab switch
  retry: 1,                     // One retry on failure
});
```

**Request deduplication** is built into TanStack Query: if multiple dashboard components request `["academic-calendar", "current"]` simultaneously, only one network fetch is fired.

### Mutation Hook

```typescript
const mutation = useSaveAcademicCalendar();

// On success:
// 1. Toast notification ("Academic calendar saved!")
// 2. queryClient.invalidateQueries(["academic-calendar", "current"])
// 3. The decision tree re-evaluates → form collapses, dashboard unlocks
```

### API Client

```typescript
// lib/api/academic-calendar.ts
fetchCurrentCalendar(): Promise<AcademicYear | null>
  // GET /api/v1/schools/current-calendar
  // Returns null on 404 (no calendar yet)

saveAcademicCalendar(payload): Promise<AcademicYear>
  // POST /api/v1/schools/current-calendar
```

---

## Component State Decision Tree

The `useCalendarEvaluator()` hook evaluates the API response against local machine time to determine what to render:

```
                       ┌─────────────┐
                       │ API returns  │
                       │ data or null?│
                       └──────┬──────┘
                              │
                    ┌─────────┴─────────┐
                    ▼                   ▼
               null / empty        has periods
                    │                   │
                    ▼                   ▼
              ┌──────────┐     ┌────────────────┐
              │  CASE 1  │     │ Evaluate dates  │
              │  "form"  │     │ against now()   │
              │  setup   │     └───────┬────────┘
              └──────────┘             │
                              ┌────────┼────────┐
                              ▼        ▼        ▼
                         now in    now <   now >
                        [start,   start   final_end
                        final_end]          │
                              │        │     ▼
                              │        │ ┌──────────┐
                              │        │ │  CASE 4  │
                              │        │ │  "form"  │
                              │        │ │next-cycle│
                              │        │ └──────────┘
                              │        ▼
                              │  ┌──────────────┐
                              │  │  CASE 3      │
                              │  │  "hidden"    │
                              │  │  prep-mode   │
                              │  └──────────────┘
                              ▼
                    ┌──────────────────┐
                    │    CASE 2        │
                    │  "hidden"        │
                    │  (dashboard      │
                    │   unlocked)      │
                    └──────────────────┘
```

| Case | Condition | Render | Dashboard State |
|---|---|---|---|
| **1** | API returns `null` or empty periods array | `AcademicCalendarForm` (setup mode) | Locked (blurred) |
| **2** | `now()` is within `[periods[0].start_date … finalPeriod.end_date]` | **Nothing** (collapsed) | **Unlocked** |
| **3** | `now()` < `periods[0].start_date` | `PrepModeAlert` strip | Locked |
| **4** | `now()` > `finalPeriod.end_date` | `AcademicCalendarForm` (next-cycle mode) | Locked |

### Edge Cases

- **Between-period gaps** (e.g., April break): The decision tree checks the *overall* year range `[first_start … final_end]`. If today falls inside that range, the form stays hidden even if it's a break week.
- **No explicit `is_final` period**: Falls back to the last period in the array.
- **Loading state**: Calendar data is fetched via TanStack Query — a loading skeleton is shown while the fetch is in flight.

---

## Form Component (State A)

### Visual Layout

```
┌──────────────────────────────────────────────────────────────┐
│ 🗓️  Set Up Academic Calendar                                 │
│                                                              │
│  Select Academic Year:   [ 2026 ]    ← centered input       │
│                                                              │
│  Define Academic Periods                     [ + Add Period ] │
│  ┌──────────────────────────────────────────────────────────┐│
│  │ [ Term 1 ] [Start Date 📅] [End Date 📅] [Set Final] [🗑]││
│  │ [ Term 2 ] [Start Date 📅] [End Date 📅] [Set Final] [🗑]││
│  │ [ Term 3 ] [Start Date 📅] [End Date 📅] [✅ Final] [🗑] ││
│  └──────────────────────────────────────────────────────────┘│
│                                                              │
│  [ Save & Activate Calendar ]  [Fill with Sample CBC Data]   │
└──────────────────────────────────────────────────────────────┘
```

### Form Fields (per period row)

| Field | Type | Control | Notes |
|---|---|---|---|
| **name** | `string` | `<Input>` | e.g. "Term 1", "Term 2" |
| **startDate** | `Date` | `<DatePicker>` | Shadcn Popover + Calendar; min/max cascaded |
| **endDate** | `Date` | `<DatePicker>` | Shadcn Popover + Calendar; min set to start + 1 day |
| **isFinal** | `boolean` | `<Button>` toggle | Radio-group behavior — only one can be `true` |

### is_final Toggle

The `is_final` field is rendered as a **visible toggle button** on each period row:

- Uses shadcn `Button` with `variant="default"` (filled) when active, `variant="outline"` when inactive.
- Shows a Lucide `Flag` icon (filled when active).
- **Radio-group behavior:** Clicking one row's toggle sets that row to `isFinal = true` and **clears all others** to `false`.
- **Default state:** The last term starts as the final one.
- **On row removal:** If the removed row was the final period, `isFinal` is reassigned to the last remaining row.

```typescript
// Radio-group logic — each row's onClick handler
onClick={() => {
  const periods = form.getValues("periods");
  periods.forEach((_, i) => {
    form.setValue(`periods.${i}.isFinal`, i === index);
  });
}}
```

### Smart Defaults

On mount, the form automatically spins up 3 rows pre-labeled `"Term 1"`, `"Term 2"`, and `"Term 3"`. The last row (`Term 3`) has `isFinal: true` by default.

### Anti-Clash Date Engine

Each date picker enforces sequential minimum date cascading:

```
Year Start (Jan 1)
     │
     ▼
Row 1 Start: min = Jan 1 of selected year
Row 1 End:   min = Row 1 Start + 1 day
     │
     ▼
Row 2 Start: min = Row 1 End + 1 day
Row 2 End:   min = Row 2 Start + 1 day
     │
     ▼
Row 3 Start: min = Row 2 End + 1 day
Row 3 End:   min = Row 3 Start + 1 day
```

All date pickers also cap the maximum selectable date to **December 31st** of the selected year.

---

## Mutation & Success Transitions

### Button Guard

The **"Save & Activate Calendar"** button remains disabled until:
- The Year input is filled (valid integer ≥ 2020).
- Every visible row has a valid name, start date, and end date.
- All period dates satisfy: `start < end` and no overlaps.

### Submission Flow

```
User clicks "Save & Activate Calendar"
  │
  ▼
Zod schema validation (client-side)
  │
  ▼ (valid)
Packages payload:
  { year, periods: [{ name, start_date, end_date, is_final }] }
  │
  ▼
useSaveAcademicCalendar().mutateAsync(payload)
  │
  ├─ Button shows loading spinner
  ├─ Form opacity drops to 60% (pointer-events: none)
  │
  ▼ (success: HTTP 200)
  │
  ├─ Form fades out
  ├─ Green CheckCircle2 icon appears (glowing animation)
  ├─ queryClient.invalidateQueries(['academic-calendar', 'current'])
  ├─ Toast: "Academic calendar saved!"
  │
  ▼ (after 1.5s)
  │
  ├─ onSuccess() callback fires
  ├─ Decision tree re-evaluates
  ├─ Top section slides upward (hidden)
  └─ Dashboard statistics & analytics unlock
```

### Loading State

```typescript
const isSubmitting = saveMutation.isPending;

// Applied to form wrapper:
<div className={`transition-all duration-300 ${
  isSubmitting ? "pointer-events-none opacity-60" : ""
}`}>
```

### Success State

```tsx
if (showSuccess) {
  return (
    <div className="flex items-center justify-center py-12">
      <div className="flex flex-col items-center gap-4 text-center">
        <CheckCircle2 className="h-16 w-16 text-emerald-500 animate-in zoom-in-50 fade-in duration-500" />
        <p className="text-lg font-medium text-emerald-700">
          Calendar activated successfully!
        </p>
      </div>
    </div>
  );
}
```

---

## Dashboard Integration

### Layout Hierarchy

```
┌──────────────────────────────────────────────────────────────┐
│  HIERARCHICAL CONTAINER ZONE (Top of viewport)               │
│  ┌──────────────────────────────────────────────────────────┐│
│  │  AcademicCalendarForm  |  PrepModeAlert  |  (hidden)     ││
│  └──────────────────────────────────────────────────────────┘│
├──────────────────────────────────────────────────────────────┤
│  TOP OF PAGE STATISTICS                                      │
│  [Total Students: --]  [Total Teachers: --]  [Active: --]    │
├──────────────────────────────────────────────────────────────┤
│  ANALYTICS WORKSPACE                                         │
│  ┌──────────────────────────────────────────────────────────┐│
│  │  Attendance Trends (chart area)                          ││
│  ├──────────────────────────────────────────────────────────┤│
│  │  Performance Assessment  │  Session Info                 ││
│  └──────────────────────────────────────────────────────────┘│
└──────────────────────────────────────────────────────────────┘
```

### Locked vs Unlocked State

When the calendar form is visible (Cases 1/4), the statistics and analytics below are visually locked:

```tsx
<div className={`transition-all duration-500 ${
  dashboardUnlocked ? "" : "pointer-events-none opacity-40 blur-sm"
}`}>
  {/* Statistics cards & analytics skeleton */}
</div>
```

When the calendar is active (Case 2), `dashboardUnlocked = true` and the lower sections are fully interactive.

### Prep Mode Alert

When today is before the academic year starts (Case 3), a minimalist amber alert strip renders:

```tsx
<div className="flex items-center gap-2 rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800">
  <span>📅</span>
  <span>System in preparation mode for upcoming year. Calendar is configured and ready.</span>
</div>
```

---

## File Reference

### Frontend

| File | Purpose |
|---|---|
| `src/features/calendar/types.ts` | Domain types, form data types, `CalendarState` union |
| `src/features/calendar/hooks/use-academic-calendar.ts` | TanStack query + mutation hooks |
| `src/features/calendar/components/academic-calendar-evaluator.tsx` | Decision tree hook + prep-mode alert |
| `src/features/calendar/components/academic-calendar-form.tsx` | State A form with anti-clash engine |
| `src/features/calendar/components/date-picker.tsx` | Shadcn date picker (Popover + Calendar) |
| `src/features/calendar/index.ts` | Barrel exports |
| `src/components/ui/calendar.tsx` | Shadcn Calendar (react-day-picker v10) |
| `src/lib/api/academic-calendar.ts` | API client (GET/POST) |
| `src/features/dashboard/components/dashboard-page.tsx` | Dashboard with evaluator integration |

### Backend

| File | Purpose |
|---|---|
| `backend/internal/academiccalendar/domain.go` | DTO types |
| `backend/internal/academiccalendar/repository.go` | SQL queries + transaction helpers |
| `backend/internal/academiccalendar/service.go` | Business logic |
| `backend/internal/academiccalendar/handler.go` | HTTP handlers + fx module |
| `backend/internal/database/migrations/000003_add_is_final_to_terms.up.sql` | Migration |
| `backend/cmd/api/main.go` | Module wiring + route registration |
