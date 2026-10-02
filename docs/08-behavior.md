# 08 — Behavior Management

## Overview
Incident logging, disciplinary actions, and parent notifications for student behavior tracking.

---

## Backend Changes

### 1. Database Tables (New Migration)
**File:** `backend/db/migrations/XXXXXX_create_behavior.up.sql`

```sql
-- Behavior categories
CREATE TABLE behavior_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    severity TEXT NOT NULL DEFAULT 'LOW', -- 'LOW', 'MEDIUM', 'HIGH', 'CRITICAL'
    color TEXT NOT NULL DEFAULT '#6366f1',
    requires_parent_notification BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (school_id, name)
);

-- Behavior incidents
CREATE TABLE behavior_incidents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES behavior_categories(id),
    reported_by UUID NOT NULL REFERENCES users(id),
    incident_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    location TEXT,
    description TEXT NOT NULL,
    witnesses JSONB DEFAULT '[]', -- [{student_id, name}]
    status TEXT NOT NULL DEFAULT 'OPEN', -- 'OPEN', 'INVESTIGATING', 'RESOLVED', 'CLOSED'
    resolution TEXT,
    resolved_by UUID REFERENCES users(id),
    resolved_at TIMESTAMPTZ,
    parent_notified BOOLEAN NOT NULL DEFAULT FALSE,
    parent_notified_at TIMESTAMPTZ,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Disciplinary actions
CREATE TABLE disciplinary_actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    incident_id UUID NOT NULL REFERENCES behavior_incidents(id) ON DELETE CASCADE,
    action_type TEXT NOT NULL, -- 'WARNING', 'DETENTION', 'SUSPENSION', 'EXPULSION', 'COUNSELING', 'PARENT_MEETING', 'OTHER'
    description TEXT,
    assigned_date DATE NOT NULL DEFAULT CURRENT_DATE,
    due_date DATE,
    completed_date DATE,
    assigned_by UUID NOT NULL REFERENCES users(id),
    supervised_by UUID REFERENCES users(id),
    status TEXT NOT NULL DEFAULT 'PENDING', -- 'PENDING', 'IN_PROGRESS', 'COMPLETED', 'CANCELLED'
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_behavior_categories_school ON behavior_categories(school_id);
CREATE INDEX idx_behavior_incidents_student ON behavior_incidents(student_id);
CREATE INDEX idx_behavior_incidents_date ON behavior_incidents(incident_date);
CREATE INDEX idx_behavior_incidents_status ON behavior_incidents(status);
CREATE INDEX idx_disciplinary_actions_incident ON disciplinary_actions(incident_id);
CREATE INDEX idx_disciplinary_actions_student ON disciplinary_actions(student_id); -- via incident
```

### 2. SQLC Queries
**File:** `backend/db/queries/behavior.sql`

### 3. Services & Handlers
- `BehaviorCategoriesService` + `Handler`
- `BehaviorIncidentsService` + `Handler`
- `DisciplinaryActionsService` + `Handler`

### 4. Router Endpoints
```
# Categories
GET    /api/behavior/categories
POST   /api/behavior/categories
PATCH  /api/behavior/categories/:id
DELETE /api/behavior/categories/:id

# Incidents
GET    /api/behavior/incidents
POST   /api/behavior/incidents
GET    /api/behavior/incidents/:id
PATCH  /api/behavior/incidents/:id
DELETE /api/behavior/incidents/:id
POST   /api/behavior/incidents/:id/notify-parents

# Actions
GET    /api/behavior/actions
POST   /api/behavior/actions
PATCH  /api/behavior/actions/:id
DELETE /api/behavior/actions/:id
```

---

## Frontend Changes

### 1. Feature Module
```
src/features/behavior/
├── components/
│   ├── incidents-table.tsx
│   ├── incident-form.tsx
│   ├── incident-detail.tsx
│   ├── actions-table.tsx
│   ├── action-form.tsx
│   └── categories-table.tsx
├── hooks/
│   ├── use-categories.ts
│   ├── use-incidents.ts
│   └── use-actions.ts
├── services/
│   └── api.ts
├── types/
│   └── behavior.ts
└── index.ts
```

### 2. Pages
```
src/app/(dashboard)/behavior/
├── page.tsx              # Incidents list
├── add/page.tsx          # Log incident
├── [id]/page.tsx         # Incident detail + actions
├── actions/page.tsx      # Disciplinary actions
└── categories/page.tsx   # Category management
```

### 3. Navigation
Already in nav: `{ title: "Behavior", url: "/behavior", icon: <AlertTriangleIcon /> }`

---

## Acceptance Criteria
- [ ] Log incident: student, category, date, location, description, witnesses
- [ ] Incident workflow: OPEN → INVESTIGATING → RESOLVED/CLOSED
- [ ] Assign disciplinary actions with due dates
- [ ] Parent notification (email/SMS - future integration)
- [ ] Student behavior history view
- [ ] Filter incidents by student, category, date, status
- [ ] Dashboard: incidents this week, by severity, by student
- [ ] RLS: tenant isolation

---

## Dependencies
- Requires: Students, Users (teachers/admins)

---

## Estimated Effort
- Backend: ~8 hours
- Frontend: ~8 hours
- **Total: ~16 hours**