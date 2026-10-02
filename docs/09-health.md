# 09 — Health & Medical

## Overview
Medical records, immunization tracking, nurse visits, and health conditions management.

---

## Backend Changes

### 1. Database Tables (New Migration)
**File:** `backend/db/migrations/XXXXXX_create_health.up.sql`

```sql
-- Health conditions (reference)
CREATE TABLE health_conditions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    is_chronic BOOLEAN NOT NULL DEFAULT FALSE,
    requires_medication BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (school_id, name)
);

-- Student health profiles
CREATE TABLE student_health_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    blood_type TEXT, -- 'A+', 'A-', 'B+', 'B-', 'AB+', 'AB-', 'O+', 'O-'
    allergies JSONB DEFAULT '[]', -- [{allergen, severity, notes}]
    medications JSONB DEFAULT '[]', -- [{name, dosage, frequency, prescribed_by}]
    conditions UUID[] DEFAULT '{}', -- health_conditions IDs
    emergency_contact_name TEXT,
    emergency_contact_phone TEXT,
    emergency_contact_relation TEXT,
    insurance_provider TEXT,
    insurance_policy_number TEXT,
    doctor_name TEXT,
    doctor_phone TEXT,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (student_id)
);

-- Immunizations
CREATE TABLE immunizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    vaccine_name TEXT NOT NULL,
    dose_number INT NOT NULL DEFAULT 1,
    total_doses INT NOT NULL DEFAULT 1,
    administered_date DATE NOT NULL,
    administered_by UUID REFERENCES users(id), -- nurse
    batch_number TEXT,
    expiry_date DATE,
    next_due_date DATE,
    status TEXT NOT NULL DEFAULT 'COMPLETED', -- 'COMPLETED', 'SCHEDULED', 'OVERDUE', 'EXEMPTED'
    exemption_reason TEXT,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Nurse visits
CREATE TABLE nurse_visits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    nurse_id UUID NOT NULL REFERENCES users(id), -- school_memberships with role NURSE
    visit_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reason TEXT NOT NULL,
    symptoms JSONB DEFAULT '[]',
    treatment TEXT,
    medication_given JSONB DEFAULT '[]',
    outcome TEXT NOT NULL, -- 'RETURNED_TO_CLASS', 'SENT_HOME', 'REFERRED_TO_DOCTOR', 'EMERGENCY'
    parent_notified BOOLEAN NOT NULL DEFAULT FALSE,
    follow_up_required BOOLEAN NOT NULL DEFAULT FALSE,
    follow_up_date DATE,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Health screenings (periodic)
CREATE TABLE health_screenings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    screening_type TEXT NOT NULL, -- 'VISION', 'HEARING', 'BMI', 'DENTAL', 'SCOLIOSIS', 'OTHER'
    screening_date DATE NOT NULL,
    conducted_by UUID REFERENCES users(id),
    results JSONB NOT NULL, -- varies by type
    referral_required BOOLEAN NOT NULL DEFAULT FALSE,
    referral_notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_student_health_profiles_student ON student_health_profiles(student_id);
CREATE INDEX idx_immunizations_student ON immunizations(student_id);
CREATE INDEX idx_immunizations_due ON immunizations(next_due_date) WHERE next_due_date IS NOT NULL;
CREATE INDEX idx_nurse_visits_student ON nurse_visits(student_id);
CREATE INDEX idx_nurse_visits_date ON nurse_visits(visit_date);
CREATE INDEX idx_health_screenings_student ON health_screenings(student_id);
```

### 2. SQLC Queries
**File:** `backend/db/queries/health.sql`

### 3. Services & Handlers
- `HealthProfilesService` + `Handler`
- `ImmunizationsService` + `Handler`
- `NurseVisitsService` + `Handler`
- `HealthScreeningsService` + `Handler`
- `HealthConditionsService` + `Handler`

### 4. Router Endpoints
```
# Health Profiles
GET    /api/health/profiles/:studentId
PATCH  /api/health/profiles/:studentId

# Immunizations
GET    /api/health/students/:studentId/immunizations
POST   /api/health/students/:studentId/immunizations
PATCH  /api/health/immunizations/:id
DELETE /api/health/immunizations/:id

# Nurse Visits
GET    /api/health/nurse-visits
POST   /api/health/nurse-visits
GET    /api/health/nurse-visits/:id
PATCH  /api/health/nurse-visits/:id

# Screenings
GET    /api/health/screenings
POST   /api/health/screenings
GET    /api/health/screenings/:id

# Conditions (reference)
GET    /api/health/conditions
POST   /api/health/conditions
PATCH  /api/health/conditions/:id
DELETE /api/health/conditions/:id
```

---

## Frontend Changes

### 1. Feature Module
```
src/features/health/
├── components/
│   ├── health-profile.tsx
│   ├── immunizations-table.tsx
│   ├── immunization-form.tsx
│   ├── nurse-visits-table.tsx
│   ├── nurse-visit-form.tsx
│   ├── screenings-table.tsx
│   └── conditions-table.tsx
├── hooks/
│   ├── use-health-profile.ts
│   ├── use-immunizations.ts
│   ├── use-nurse-visits.ts
│   └── use-screenings.ts
├── services/
│   └── api.ts
├── types/
│   └── health.ts
└── index.ts
```

### 2. Pages
```
src/app/(dashboard)/health/
├── page.tsx                      # Dashboard: upcoming immunizations, recent visits
├── students/[id]/page.tsx        # Student health profile
├── immunizations/page.tsx        # Immunization management
├── nurse-visits/page.tsx         # Nurse visit log
└── screenings/page.tsx           # Health screenings
```

### 3. Navigation
Already in nav: `{ title: "Health", url: "/health", icon: <HeartPulse /> }`
Also: `/nurses` (staff management) — separate feature

---

## Acceptance Criteria
- [ ] Student health profile: blood type, allergies, medications, conditions, emergency contacts
- [ ] Immunization tracking: schedule, record, due dates, exemptions
- [ ] Nurse visit logging: reason, symptoms, treatment, outcome, parent notification
- [ ] Health screenings: vision, hearing, BMI, dental, scoliosis
- [ ] Upcoming immunization alerts on dashboard
- [ ] Privacy: RLS + field-level encryption for sensitive fields (future)
- [ ] Nurse role: can log visits, view assigned students

---

## Dependencies
- Requires: Students, Users (nurses)
- Future: Parent portal (view health profile)

---

## Estimated Effort
- Backend: ~12 hours (5 tables, privacy considerations)
- Frontend: ~10 hours
- **Total: ~22 hours**