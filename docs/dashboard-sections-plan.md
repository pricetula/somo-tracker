# New Mock Components — Schema-Backed (No Existing Components)

Constraints:
- No bulk-jobs (`bulk_jobs`, `bulk_job_items` excluded)
- No substitution-teacher (`timetable_substitutions` excluded)
- All events in one unified section (`school_events` + `public_holidays` + `event_attendance`)
- Existing dashboard components (summary-cards, needs-attention, attendance-kpi, class-coverage-mini, classes-table, today-panel, dashboard-header) are NOT covered here

---

## 1. Streams Widget / Filter

| Source | Show | Graph / Visual | Action |
|---|---|---|---|
| `streams` (school_id, name) | Stream names (Simba, Red house) | Small filter chips / cards | Filter other widgets by stream |

New component — not mocked anywhere currently.

---

## 2. Curriculum / Subject Widget

| Source | Show | Graph / Visual | Action |
|---|---|---|---|
| `subjects`, `topics`, `sub_topics` | Subjects taught this week / today; topic coverage % | List or progress bars | — |

New component — zero curriculum presence on dashboard today.

---

## 3. Rooms Widget (Utilization / Availability)

| Source | Show | Graph / Visual | Action |
|---|---|---|---|
| `rooms` (name, capacity?, school_id) | Room usage vs capacity; unassigned rooms | Small bar / grid | View schedule |

New component — only mentioned as text "Room4" in existing mock; no data backing.

---

## 4. Unified Events Section (Single Block — Design To Discuss)

> All events + holidays in one area. Replaces scattered event mock.

| Sub-block | Source | Show | Graph / Visual | Action |
|---|---|---|---|---|
| Upcoming | `school_events` (title, event_type, start/end, requires_attendance) | List / cards | — | View |
| Past / current attendance | `event_attendance` | Attendance % for events requiring it | Small % bar | Mark |
| Public holidays | `public_holidays` (date, name) | Holiday badges / banner near events | Calendar pill | — |

New design — existing `upcoming-events` mock is hardcoded; this is the redesigned unified block with `public_holidays` included.

---

## 5. Guardian-Student Link Health

| Source | Show | Graph / Visual | Action |
|---|---|---|---|
| `guardian_student_links` | Unlinked students / guardians; coverage ratio | Small metric / alert | Link / remind |

New component — summary-cards only shows gross guardian count.

---

## 6. Enrollment / Capacity Widget

| Source | Show | Graph / Visual | Action |
|---|---|---|---|
| `student_class_enrollments` + `grade_levels` | Enrolled vs capacity per grade/stream/term | Progress bars | Adjust |

New component — only mocked as raw counts in existing cards.

---

## 7. Grade / Education System Context

| Source | Show | Graph / Visual | Action |
|---|---|---|---|
| `grade_levels`, `education_systems` | Filter labels / headers (PP1, Grade 1, CBE…) | Small tag / dropdown | Filter |

New — currently only text strings in existing mocks.

---

## Exclusions (Not Documented Here)

- `bulk_jobs` / `bulk_job_items` → excluded entirely
- `timetable_substitutions` → excluded entirely
- Existing components (summary-cards, needs-attention, attendance-kpi, class-coverage-mini, today-panel, classes-table, dashboard-header) → not covered
- Fee/transaction widgets → blocked; no `fees` table in schema

---

## Open Design Questions

1. How does the unified events section look? (list, cards, calendar?)
2. Where do public holiday badges sit — top of events, or header?
3. Do streams filter the whole dashboard or get their own card?
4. Curriculum widget: weekly view or topic-level detail?
5. Room widget: usage % per room, or availability grid?
