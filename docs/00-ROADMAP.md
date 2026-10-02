# Feature Implementation Roadmap — Somotracker

> Numbered specs for sequential implementation. Start at 01, work through to 10.

---

## Phase 1: Core Timetable Foundations (Week 1-2)

| # | Spec | Est. Hours | Status |
|---|------|------------|--------|
| 01 | [Rooms CRUD](01-rooms-crud.md) | 7 | 📋 Ready |
| 02 | [Timetable Substitutions](02-timetable-substitutions.md) | 9 | 📋 Ready |
| 03 | [Student Enrollments](03-student-enrollments.md) | 11 | 📋 Ready |

**Phase 1 Total: ~27 hours**

---

## Phase 2: Major Pedagogical Features (Week 3-6)

| # | Spec | Est. Hours | Status |
|---|------|------------|--------|
| 04 | [Assessments Module](04-assessments.md) | 38 | 📋 Ready |

**Phase 2 Total: ~38 hours** (largest single feature)

---

## Phase 3: Calendar & Events (Week 7-8)

| # | Spec | Est. Hours | Status |
|---|------|------------|--------|
| 05 | [Event Timeline & Public Holidays](05-event-timeline.md) | 20 | 📋 Ready |

**Phase 3 Total: ~20 hours**

---

## Phase 4: Finance (Week 9-11)

| # | Spec | Est. Hours | Status |
|---|------|------------|--------|
| 06 | [Finance Module](06-finance.md) | 33 | 📋 Ready |

**Phase 4 Total: ~33 hours**

---

## Phase 5: Analytics & Student Services (Week 12-14)

| # | Spec | Est. Hours | Status |
|---|------|------------|--------|
| 07 | [Reports & Analytics](07-reports.md) | 27 | 📋 Ready |
| 08 | [Behavior Management](08-behavior.md) | 16 | 📋 Ready |
| 09 | [Health & Medical](09-health.md) | 22 | 📋 Ready |

**Phase 5 Total: ~65 hours**

---

## Phase 6: Settings Polish (Week 15)

| # | Spec | Est. Hours | Status |
|---|------|------------|--------|
| 10 | [Settings Completion](10-settings-completion.md) | 11 | 📋 Ready |

**Phase 6 Total: ~11 hours**

---

## Grand Total: ~194 hours (~5-6 weeks for 1 dev)

---

## Quick Reference: Navigation Coverage

| Nav Item | Spec | Implemented |
|----------|------|-------------|
| Dashboard | — | ✅ |
| Members | — | ✅ |
| Curriculum | — | ✅ |
| Classes | — | ✅ |
| Time table | 01, 02, 03 | ⚠️ Partial |
| Attendance | — | ✅ |
| **Assessments** | **04** | ❌ |
| School Events | 05 | ⚠️ Partial |
| **Reports** | **07** | ❌ |
| **Behavior** | **08** | ❌ |
| **Finance** | **06** | ❌ |
| **Health** | **09** | ❌ |
| Settings | 10 | ⚠️ Partial |

---

## Implementation Notes

### Ordering Rationale
1. **Phase 1** unblocks timetable (rooms, substitutions, enrollments)
2. **Phase 2** delivers highest-value pedagogical feature (assessments)
3. **Phase 3** completes calendar (design doc ready)
4. **Phase 4** revenue-critical (finance)
5. **Phase 5** builds on core data (reports needs assessments/finance)
6. **Phase 6** admin polish

### Parallelization Opportunities
- 01, 02, 03 can be worked in parallel (different domains)
- 08, 09 can be parallel (independent student services)
- 07 should wait for 04, 06 data

### Prerequisites
| Spec | Requires |
|------|----------|
| 02 | 01 (rooms for timetable) |
| 03 | Classes, Students, Academic Years |
| 04 | 03 (enrollments), Timetable slots |
| 05 | Countries table |
| 06 | 03 (enrollments for invoicing) |
| 07 | 04, 06, Attendance |
| 08, 09 | Students |
| 10 | Education Systems, Countries |

---

## Getting Started

```bash
# Start with Phase 1 - Rooms CRUD
# 1. Create migration: backend/db/migrations/XXXXXX_create_rooms.up.sql
# 2. Add SQLC queries: backend/db/queries/rooms.sql
# 3. Run: make sqlc-gen
# 4. Implement service, handler, router
# 5. Frontend: src/features/rooms/ + page
# 6. Test: backend tests + frontend build
```

---

## Related Documents
- `docs/features-to-build.md` — Full feature audit
- `docs/tables-endpoints-queries.md` — Schema/endpoint inventory
- `docs/event-timeline-ui-plan.md` — Timeline UX design
- `ATTENDANCE_FEATURE_PLAN.md` — Implementation pattern reference