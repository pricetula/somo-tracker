Scratch DB cleaned up. Here's the full audit.

 ────────────────────────────────────────────────────────────────────────────────

 Deep Audit: 000001_initial_schema.up.sql

 Scope: 6,541 lines, 57 tables, 199 index statements, 29 functions, 66 triggers, 25 enums, 190 comments. Applied cleanly on Postgres 16 (up and down) — no syntax or ordering errors, and the project's
 TestMigrationStaticAnalysis passes.

 The good stuff first: the (tenant_id, id) composite-FK pattern for tenant scoping, per-event trigger splitting (insert/delete/update with transition tables), partial unique indexes for soft-delete workflows,
 GiST exclusion constraints on timetable/ranges, and extensive COMMENTs are all genuinely well done.

 But I found 2 critical bugs, 2 critical security gaps, and a stack of correctness/performance issues — several verified with live reproduction against Postgres.

 ────────────────────────────────────────────────────────────────────────────────

 🔴 CRITICAL

 ### C1. The "weighted exam score" is mathematically a no-op — KNEC weights never apply

 fn_compute_term_overall_summaries_for_term, weighted branch. The subject-to-weight join has no key linking assessment_weight_configs to learning areas — it joins resolved_weight_configs to subject_summaries
 on student_id only:

 ```sql
   SELECT ROUND(SUM(ss2.average_percentage * rwc2.weight_percent) /
                NULLIF(SUM(rwc2.weight_percent), 0)::NUMERIC, 2)
   FROM subject_summaries ss2
   JOIN resolved_weight_configs rwc2 ON rwc2.student_id = ss2.student_id
 ```

 Every subject row multiplies against every config row, so the sums factor:

 │ numerator = (Σw) × (Σp), denominator = n_subjects × (Σw) → Σp / n_subjects

 The weights always cancel out — the output is exactly the plain average, every time. Verified live: G9 student with Math=80, English=60, final term, 60% KJSEA config → result 70.00 with
 is_weighted_exam_score = true. So report cards display a "weighted KNEC score" that is actually just the unweighted mean, and the flag is misleading. The seed data and comments document real formulas (KPSEA
 60/40, KJSEA 20/20/60) that the code can never produce.

 Fix: this needs a real mapping — either an assessment_type_code ↔ learning_area.code link or an explicit weight_scope/learning_area_id column on the config table — then compute per-subject weighted
 contributions.

 ### C2. RLS is completely inert — the app connects as a superuser with BYPASSRLS

 The entire RLS layer (40+ tables, fn_current_tenant_id, SET LOCAL app.current_tenant_id) is decorative. The app's DB role, somo_admin, is Superuser, ... Bypass RLS and is the table owner. ENABLE ROW LEVEL
 SECURITY without FORCE ROW LEVEL SECURITY doesn't restrict the owner or superusers.

 So the "second line of defence" the header comment advertises does not exist: one missed WHERE tenant_id = ... in the app = cross-tenant data leak, which is precisely what RLS was built to prevent. Verified:
 a limited role (which should be filtered to zero rows via the policy) reads all rows unless FORCE is set and the role isn't an owner.

 Fix: create a dedicated non-owner app role, GRANT per-table privileges, ALTER TABLE ... FORCE ROW LEVEL SECURITY on the tenant tables, and set app.current_tenant_id per transaction. Without this, the whole
 RLS section is dead weight with a false sense of security.

 ### C3. Rubric → percentage mapping leaks across grading profiles and tenants

 fn_refresh_term_subject_summary_for_session joins grading_scale_ranges on performance_level only — no profile_id, no school_id, no tenant_id filter:

 ```sql
   LEFT JOIN grading_scale_ranges r
       ON r.performance_level = sog.awarded_level
       AND r.default_percentage_mapping IS NOT NULL
 ```

 A rubric grade maps to every matching range in every profile of every school. Verified live: one ME outcome grade, active profile maps 65.00, retired profile maps 75.00 → summary wrote average_percentage =
 70.00 (the mean of both). Wrong averages, and tenant B's grading profiles silently influence tenant A's reports. The header comment even promises "first matching range from any active profile belonging to
 the school" — the SQL does none of that.

 Fix: scope the join to the session's school/active profile (like fn_refresh_subject_strand_summary_for_session correctly does with v_scale_profile_id).

 ### C4. sessions (and class_daily_attendance_summaries) have tenant_id but no RLS

 sessions holds stytch_session_token and token_hash — yet it's absent from both the ENABLE ROW LEVEL SECURITY list and the policy loops. Same for class_daily_attendance_summaries. Even after fixing C2, these
 two stay unprotected. This is the most security-sensitive table in the schema.

 ────────────────────────────────────────────────────────────────────────────────

 🟠 HIGH

 ### H1. Payments trigger does a full-table aggregate on every insert

 fn_sync_invoice_payment_status_insert/delete/update:

 ```sql
   LEFT JOIN (SELECT invoice_id, SUM(amount) FROM payments GROUP BY invoice_id) p ...
 ```

 No filter on the affected invoice — every single payment write scans and aggregates the entire payments table. On the hot financial path this is O(n) per insert, O(n²) overall. Verified with EXPLAIN: adding
 WHERE invoice_id IN (affected ids) turns it into an index-only scan via idx_payments_invoice_id.

 Fix: WHERE invoice_id IN (SELECT invoice_id FROM inserted_rows/deleted_rows) inside the aggregate subquery.

 ### H2. academic_years.name::INT cast in 4 functions

 fn_compute_term_overall_summaries_for_term, fn_compute_single_student_term_overall_summary, fn_compute_performance_projections_for_term, fn_compute_teacher_subject_performance_summaries all do ay.name::INT.
 If any school names a year "2024/25", "AY2024", or "Year 2024", these batch jobs fail at runtime. Arithmetic is implicitly coupled to a display string. Add a real integer column (e.g. start_year).

 ### H3. RLS default-deny dead ends: cbc_student_parents and school_member_counts

 Both are RLS-enabled but have no tenant_id column → the policy loop correctly skips them → for any non-owner role they're completely unreadable/unwritable (default deny). The comment calls this "safe by
 default" but it means: the moment C2 is fixed, parent–student linking and member counts silently return zero rows with no policy in place. cbc_student_parents also uses single-column FKs (student_id,
 parent_id) with no tenant scoping — cross-tenant parent↔student links are possible at the FK level.

 ### H4. Dead code left over from the squash

 - tmp_weight_configs temp table created, never used (the comment claims it caches configs; the query reads the table directly).
 - v_target_exam assigned CASE WHEN v_is_final THEN NULL ELSE NULL END — a tautology that's never referenced.
 - v_weight_total declared, never used.
 - behavior_category_type enum created twice (lines 197 and 5030) with a duplicated COMMENT.
 - block_type enum defined, used nowhere.
 - fn_rls_tenant_policy() returns a policy TEXT that no CREATE POLICY ever calls.

 ────────────────────────────────────────────────────────────────────────────────

 🟡 MEDIUM

 ### M1. Redundant indexes duplicating unique constraints (write amplification)

 - idx_tenants_slug duplicates UNIQUE(slug)
 - idx_tenants_stytch_org_id duplicates UNIQUE(stytch_org_id)
 - idx_cbc_classes_school_year_grade_stream duplicates uq_cbc_classes_tier_stream — identical 4-column key, same order

 Each costs an extra index write on every insert. Drop the explicit ones (the constraint already creates the index).

 ### M2. grading_scale_profiles deletion is a confusing trap

 assessment_sessions.grading_scale_profile_id ... ON DELETE SET NULL combined with chk_quantitative_has_scale (QUANTITATIVE ⇒ scale required). Deleting a referenced profile either raises a misleading P0002
 (via the ranges' immutability trigger firing on the CASCADE delete — an unintended interaction) or, for a profile with no ranges, a 23514 from the check constraint rejecting the SET NULL. Profiles are
 effectively undeletable either way, but the error messages describe neither the actual rule nor the intended one. Consider ON DELETE RESTRICT with a real message, or drop the SET NULL + document it.

 ### M3. Inconsistent tenant scoping on FKs

 The schema establishes the (tenant_id, id) composite-FK pattern but uses plain single-column FKs in many places: attendance_records.timetable_slot_id, behavior_notes.timetable_slot_id/category_id,
 assessment_sessions.learning_area_id, student_assessment_outcome_grades.performance_indicator_id, cbc_attendance_sessions.timetable_slot_id, medical_incidents.logged_by, payments.recorded_by,
 sessions.user_id, invitations.invited_by, import_jobs.created_by, and all the summary tables' learning_area_id/strand_id/sub_strand_id. RLS would mitigate this — but it doesn't run (C2). A buggy app can
 insert cross-tenant references today.

 ### M4. chk_score_range calls a STABLE function that reads another table

 max_points_check(session_id, raw_score) in a CHECK constraint works, but it makes constraint validation depend on live data in assessment_sessions, complicates pg_dump/restore, and silently weakens (becomes
 a tautology) when max_points is NULL. A trigger-based check or a NOT NULL max_points enforcement would be more robust.

 ### M5. Missing indexes on FK/query columns

 behavior_notes(category_id), behavior_notes(reviewed_by_id), payments(recorded_by), medical_incidents(logged_by), invitations(invited_by), cbc_class_teachers(learning_area_id), attendance_records(marked_by)
 — all FK columns with ON DELETE SET NULL/RESTRICT, so user/category deletes do full scans. Also no index backing uq_invitations_school_email_pending-style lookups beyond the partial unique (which covers it).

 ### M6. fn_current_tenant_id() swallows every exception

 EXCEPTION WHEN OTHERS THEN RETURN NULL masks misconfiguration — a broken app.current_tenant_id value silently returns NULL and RLS (when it works) returns zero rows with no error. The app can't distinguish
 "no tenant set" from "garbage setting". Let the cast error propagate.

 ────────────────────────────────────────────────────────────────────────────────

 🟢 LOW / CONSISTENCY

 - fee_categories: no created_at (every other table has it), no is_active (deletion hard-cascades into fee_templates/invoice_items, destroying history) — inconsistent with the soft-delete pattern used
   everywhere else.
 - Missing created_at: performance_indicators, cbc_strands, cbc_sub_strands, cbc_learning_areas (though they have updated_at).
 - chk_cbc_student_gender CHECK (gender IN ('M','F')) is redundant — the gender_type enum already restricts values.
 - Room double-booking in cbc_timetable_slots: unique_room_slot only fires for non-NULL room_identifier — rooms without identifiers can be double-booked by design (maybe intended, but undocumented).
 - sessions.token/token_hash both nullable with no guarantee at least one is set; device_fingerprint and stytch_session_token stored plaintext (the schema itself flags the hashing TODO — it's a standing
   security debt).
 - import_job_failures.raw_payload / import_job_chunks hold potentially-PII payloads with no RLS at all.
 - Comment drift: the header says "squashed migrations 000003–000017", the materialised section says "000005–000016"; inline "Migration: 0000xx_create_..." headers reference a migration numbering that no
   longer exists as separate files.
 - Seed: hardcoded demo KNEC code 12345678 — if the seed is ever run against a real school database, the unique index on knec_school_code will collide.

 ────────────────────────────────────────────────────────────────────────────────

 Suggested priority order

 1. C2 — RLS enforcement (dedicated role + FORCE ROW LEVEL SECURITY). Biggest actual risk: cross-tenant leaks are possible today.
 2. C1 — weighted score math (parent-facing report cards show wrong numbers).
 3. C3 — profile-scoped rubric mapping (wrong averages + cross-tenant influence).
 4. C4 / H3 — close RLS coverage gaps (sessions, daily summaries, junction tables).
 5. H1 — payments trigger full scan.
 6. H2, H4, M1, M2 — robustness/cleanup.

 Want me to implement any of these fixes? I'd suggest starting with C1 (requires a design decision on the config↔subject mapping) and C2 (requires touching the compose file / role provisioning), and I can
 draft the patches for the rest — H1, H2, H4, M1 are self-contained SQL changes.
