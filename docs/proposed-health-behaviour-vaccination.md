# Proposed schema additions — Health, Behaviour, Vaccination

Status: proposal / ideation phase. Mock components (`HealthBehaviourPanel`, `MisbehaviourPanel`) already implemented; these tables back them.

---

## health_notes

| Column | Type | Notes |
|---|---|---|
| id | UUID PK | |
| school_id | UUID FK → schools | Scoped per school |
| student_id | UUID FK → students | |
| note_type | VARCHAR | AILMENT / FIRST_AID / MEDICATION / VACCINATION / CHECKUP |
| description | TEXT | Free-text log |
| recorded_by | VARCHAR | Nurse / teacher / admin |
| recorded_at | TIMESTAMPTZ | |
| status | VARCHAR | ACTIVE / RESOLVED |

**Relationships:** `student_id → students(student_id)`; `school_id → schools(id)`.

---

## behaviour_flags

| Column | Type | Notes |
|---|---|---|
| id | UUID PK | |
| school_id | UUID FK → schools | |
| student_id | UUID FK → students | |
| flag_type | VARCHAR | NOISE / FIGHTING / DISRUPTION / POSITIVE / REPEATED |
| severity | VARCHAR | LOW / MEDIUM / HIGH |
| description | TEXT | |
| recorded_by | VARCHAR | |
| recorded_at | TIMESTAMPTZ | |
| status | VARCHAR | OPEN / PARENT_NOTIFIED / MEETING_SCHEDULED / RESOLVED |

**Relationships:** `student_id → students(student_id)`.

---

## behaviour_notes (incident detail)

| Column | Type | Notes |
|---|---|---|
| id | UUID PK | |
| behaviour_flag_id | UUID FK → behaviour_flags | |
| incident_time | TIMESTAMPTZ | When it happened |
| witness_count | INT | |
| action_taken | VARCHAR | What staff did |
| parent_contacted | BOOLEAN | |

---

## vaccination_records

| Column | Type | Notes |
|---|---|---|
| id | UUID PK | |
| school_id | UUID FK → schools | |
| student_id | UUID FK → students | |
| vaccine_type | VARCHAR | e.g. MMR, Polio, Tetanus |
| date_given | DATE | |
| status | VARCHAR | COMPLETED / OVERDUE / EXCUSED |

---

## Optional / future

- `parent_communication_logs` — SMS/email/phone to guardian; links to `behaviour_flags.status = PARENT_NOTIFIED`
- `assessment_scores` — exam / term grades; links to `SchoolEvent` (exam events)
- `fee_transactions` — payments; links to fee KPI / trend

---

No migrations committed; all changes remain mock / proposal until schema freeze reviewed by platform team.
