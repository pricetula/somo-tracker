### Assessments

**assessment_student_term_summary** → **Radar Chart** if broken out by subject (needs joining with the class-level table below for per-subject axes), or a single **Line Chart** point on a student's progress-over-terms view.

- Data shape: { subject: "Math", level: "ME" }, { subject: "English", level: "EE" }... (radar needs per-subject rows, which actually come from the class-level table filtered to one student's class+subjects — this table gives the overall number)
- Table: assessment_student_term_summary — student_id, academic_term_id, mean_percentage, ee_count, me_count, ae_count, be_count
- Relationships: student_id → cbc_students

**assessment_class_learning_area_term_summary** → **Bar Chart** (grouped/stacked by performance level) for "subject strength across the class," or a **Heatmap-style Bar Chart** (subjects × classes) for a school-wide subject comparison.

- Data shape: { subject: "Math", EE: 5, ME: 20, AE: 8, BE: 2 } per subject, or { class: "6-Blue", meeting_expectation_rate: 78 } across classes for one subject
- Table: assessment_class_learning_area_term_summary — class_id, learning_area_id, academic_term_id, mean_percentage, ee_count, me_count, ae_count, be_count, meeting_expectation_rate
- Relationships: class_id → cbc_classes, learning_area_id → cbc_learning_areas (has name, grade_level)

**assessment_school_term_trend** → **Line Chart** for the school-wide academic trend/projection.

- Data shape: { term: "2026 T1", mean_percentage: 72, meeting_expectation_rate: 68 }
- Table: assessment_school_term_trend — school_id, academic_term_id, term_number, academic_year_id, mean_percentage, meeting_expectation_rate

### Health

**health_incident_term_summary** → **Bar Chart** (one bar per term) as a KPI strip, or feed into a single **Radial/stat card**.

- Data shape: { term: "Term 1", incidents: 34, unique_students: 22, per_100: 6.4 }
- Table: health_incident_term_summary — school_id, academic_term_id, total_incidents, unique_students_affected, incidents_per_100_students

**health_incident_monthly_trend** → **Line Chart** for seasonal spikes (flu season etc.), independent of term boundaries.

- Data shape: { month: "2026-01", incidents: 12 }, { month: "2026-02", incidents: 25 }...
- Table: health_incident_monthly_trend — school_id, month_start, incident_count, unique_students

**health_risk_profile_summary** → **Pie Chart** or **Radial Chart** for "% of students with an allergy/chronic condition on file."

- Data shape: { category: "Allergies", count: 40 }, { category: "Chronic conditions", count: 15 }, { category: "None on file", count: 340 }
- Table: health_risk_profile_summary — school_id, students_with_allergies, students_with_chronic_conditions, students_with_profile

### Finance

**finance_term_collection_summary** → **Bar Chart** (billed vs. collected side-by-side) or a **Radial Chart** for the single collection-rate % as the headline KPI number.

- Data shape: { term: "Term 1", billed: 4200000, collected: 3650000, rate: 86.9 }
- Table: finance_term_collection_summary — school_id, academic_term_id, total_billed, total_collected, outstanding_balance, collection_rate_percentage, invoices_paid, invoices_partial, invoices_unpaid, invoices_waived

**finance_grade_term_summary** → **Bar Chart** (horizontal, one bar per grade) for "which grades are defaulting."

- Data shape: { grade: "G4", billed: 400000, collected: 310000, rate: 77.5 }
- Table: finance_grade_term_summary — school_id, academic_term_id, grade_level, total_billed, total_collected, collection_rate_percentage

**finance_fee_category_term_summary** → **Pie Chart** for revenue mix (tuition vs. transport vs. activity fees).

- Data shape: { category: "Tuition", collected: 2800000 }, { category: "Transport", collected: 400000 }...
- Table: finance_fee_category_term_summary — fee_category_id, academic_term_id, total_billed, total_collected, collection_rate_percentage
- Relationships: fee_category_id → fee_categories (has name)

**finance_daily_collections_trend** → **Area Chart** for daily cash-flow, M-Pesa reconciliation view.

- Data shape: { date: "2026-01-05", total: 85000, count: 12 }
- Table: finance_daily_collections_trend — school_id, date, payments_total, payments_count, avg_payment

### Members / Enrollment

**member_growth_snapshots** → **Line Chart** (multi-series: teachers, students, parents as separate lines) for headcount growth over time.

- Data shape: { date: "2026-01-01", students: 340, teachers: 22, parents: 610 }
- Table: member_growth_snapshots — school_id, snapshot_date, admins, teachers, nurses, finance, parents, students

**enrollment_term_summary** → **Bar Chart** (stacked: admissions, transfers in/out, suspensions) — an enrollment funnel view.

- Data shape: { term: "Term 1", new: 15, transferred_in: 3, transferred_out: 5, suspended: 2, closing: 345 }
- Table: enrollment_term_summary — school_id, academic_term_id, opening_count, new_admissions, transferred_in, transferred_out, suspended_count, closing_count, net_change

**enrollment_class_term_summary** → **Bar Chart** (grouped, male/female per class) for a gender-balance-by-class view.

- Data shape: { class: "6-Blue", male: 18, female: 20 }
- Table: enrollment_class_term_summary — class_id, academic_term_id, grade_level, enrolled_count, male_count, female_count
