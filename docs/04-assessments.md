# 04 — Assessments Module

## Overview
Full assessment system: assessment definitions, sessions, grading scales, weight configurations, and result entry. Major pedagogical feature promised in navigation.

---

## Backend Changes

### 1. Database Tables (New Migration)
**File:** `backend/db/migrations/XXXXXX_create_assessments.up.sql`

```sql
-- Grading scales (e.g., A-F, 1-100, Pass/Fail)
CREATE TABLE grading_scales (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    scale_type TEXT NOT NULL, -- 'LETTER', 'NUMERIC', 'PASS_FAIL', 'CUSTOM'
    min_score NUMERIC,
    max_score NUMERIC,
    passing_score NUMERIC,
    bands JSONB NOT NULL DEFAULT '[]', -- [{label, min, max, color}]
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (school_id, name)
);

-- Assessment weight configurations
CREATE TABLE assessment_weight_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    weights JSONB NOT NULL, -- {"formative": 0.4, "summative": 0.6}
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (school_id, name)
);

-- Assessment definitions (reusable templates)
CREATE TABLE assessments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    class_timetable_slot_id UUID REFERENCES class_timetable_slots(id) ON DELETE SET NULL,
    title TEXT NOT NULL,
    description TEXT,
    assessment_type TEXT NOT NULL, -- 'FORM', 'SUMM', 'DIAG', 'PROJ', 'PRACT'
    max_score NUMERIC NOT NULL DEFAULT 100,
    weight_config_id UUID REFERENCES assessment_weight_configs(id) ON DELETE SET NULL,
    grading_scale_id UUID REFERENCES grading_scales(id) ON DELETE SET NULL,
    due_date DATE,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Assessment sessions (instances for a class/date)
CREATE TABLE assessment_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    assessment_id UUID NOT NULL REFERENCES assessments(id) ON DELETE CASCADE,
    class_room_id UUID NOT NULL REFERENCES class_rooms(id) ON DELETE CASCADE,
    session_date DATE NOT NULL,
    status TEXT NOT NULL DEFAULT 'SCHEDULED', -- 'SCHEDULED', 'IN_PROGRESS', 'COMPLETED', 'CANCELLED'
    conducted_by UUID REFERENCES users(id),
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (assessment_id, class_room_id, session_date)
);

-- Assessment results (student scores)
CREATE TABLE assessment_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    assessment_session_id UUID NOT NULL REFERENCES assessment_sessions(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    score NUMERIC,
    grade TEXT, -- computed from grading scale
    is_graded BOOLEAN NOT NULL DEFAULT FALSE,
    graded_by UUID REFERENCES users(id),
    graded_at TIMESTAMPTZ,
    feedback TEXT,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (assessment_session_id, student_id)
);

-- Indexes
CREATE INDEX idx_assessments_school ON assessments(school_id);
CREATE INDEX idx_assessment_sessions_assessment ON assessment_sessions(assessment_id);
CREATE INDEX idx_assessment_sessions_class_date ON assessment_sessions(class_room_id, session_date);
CREATE INDEX idx_assessment_results_session ON assessment_results(assessment_session_id);
CREATE INDEX idx_assessment_results_student ON assessment_results(student_id);
```

### 2. SQLC Queries
**File:** `backend/db/queries/assessments.sql` (new) — CRUD for all 5 tables

### 3. Services
**Files:** 
- `backend/internal/services/assessments_service.go` — Assessment CRUD
- `backend/internal/services/grading_scales_service.go` — Grading scale CRUD
- `backend/internal/services/weight_configs_service.go` — Weight config CRUD
- `backend/internal/services/assessment_sessions_service.go` — Session management
- `backend/internal/services/assessment_results_service.go` — Result entry/bulk

### 4. Handlers
**Files:**
- `backend/internal/api/assessments_handler.go`
- `backend/internal/api/grading_scales_handler.go`
- `backend/internal/api/weight_configs_handler.go`
- `backend/internal/api/assessment_sessions_handler.go`
- `backend/internal/api/assessment_results_handler.go`

### 5. Router Endpoints
```
GET    /api/assessments
POST   /api/assessments
GET    /api/assessments/:id
PATCH  /api/assessments/:id
DELETE /api/assessments/:id

GET    /api/assessments/grading-scales
POST   /api/assessments/grading-scales
PATCH  /api/assessments/grading-scales/:id
DELETE /api/assessments/grading-scales/:id

GET    /api/assessments/weight-configs
POST   /api/assessments/weight-configs
PATCH  /api/assessments/weight-configs/:id
DELETE /api/assessments/weight-configs/:id

GET    /api/assessments/sessions
POST   /api/assessments/sessions
GET    /api/assessments/sessions/:id
PATCH  /api/assessments/sessions/:id

POST   /api/assessments/results/bulk
GET    /api/assessments/sessions/:id/results
```

---

## Frontend Changes

### 1. Feature Module
```
src/features/assessments/
├── components/
│   ├── assessments-table.tsx
│   ├── assessment-form.tsx
│   ├── sessions-table.tsx
│   ├── session-form.tsx
│   ├── grading-scales-table.tsx
│   ├── grading-scale-form.tsx
│   ├── weight-configs-table.tsx
│   ├── weight-config-form.tsx
│   └── results-entry.tsx
├── hooks/
│   ├── use-assessments.ts
│   ├── use-grading-scales.ts
│   ├── use-weight-configs.ts
│   ├── use-sessions.ts
│   └── use-results.ts
├── services/
│   └── api.ts
├── types/
│   └── assessment.ts
└── index.ts
```

### 2. Pages
```
src/app/(dashboard)/assessments/
├── page.tsx                    # Assessments list
├── add/page.tsx                # Create assessment
├── [id]/page.tsx               # Assessment detail
├── sessions/page.tsx           # Sessions list
├── grading-scales/page.tsx     # Grading scales
├── weight-configs/page.tsx     # Weight configs
└── results/page.tsx            # Results entry
```

### 3. Navigation
Already in `nav-main.tsx`:
```tsx
{
    title: "Assessments",
    url: "#",
    icon: <ClipboardCheckIcon />,
    items: [
        { title: "Sessions", url: "/assessments" },
        { title: "Grading Scales", url: "/assessments/grading-scales" },
        { title: "Weight Configs", url: "/assessments/weight-configs" },
    ],
}
```

---

## Acceptance Criteria
- [ ] Create assessment with type, max score, weight config, grading scale
- [ ] Create grading scales (letter, numeric, pass/fail, custom bands)
- [ ] Create weight configurations
- [ ] Schedule assessment sessions for classes
- [ ] Bulk enter results (spreadsheet-like grid)
- [ ] Auto-compute grades from grading scale
- [ ] Filter sessions by class, date, status
- [ ] Student result view (transcript-ready)
- [ ] RLS: tenant isolation on all tables

---

## Dependencies
- Requires: Classes, Timetable slots, Students, Grading scales, Weight configs
- Blocks: Reports/Analytics (needs assessment data)

---

## Estimated Effort
- Backend: ~20 hours (5 tables, complex relationships)
- Frontend: ~18 hours (5 pages, results grid)
- **Total: ~38 hours** (largest single feature)