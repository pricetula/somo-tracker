/**
 * Assessments API — Grading Scale Profiles, Assessment Sessions, Scores & Report Cards.
 *
 * Endpoints:
 *   Grading Profiles:     POST/GET/GET/:id/PUT/:id/toggle/DELETE /api/v1/grading/profiles
 *   Grading Ranges:       POST/GET/DELETE /api/v1/grading/profiles/:id/ranges
 *   Assessment Sessions:  POST/GET/GET/:id /api/v1/assessments/sessions
 *   Session Workflow:     POST /api/v1/assessments/sessions/:id/{submit,approve,reject}
 *   Scores:               POST/GET /api/v1/assessments/sessions/:id/{scores,grades}
 *   Parent View:          GET /api/v1/parent/students/:studentId/{assessments,report-card}
 */

import { api } from "./client";

// ════════════════════════════════════════════════════════════════════════════
// TYPES
// ════════════════════════════════════════════════════════════════════════════

export interface ScaleProfile {
    id: string;
    name: string;
    is_active: boolean;
    created_at: string;
    updated_at: string;
}

export interface ScaleRange {
    id: string;
    profile_id: string;
    performance_level: "EE" | "ME" | "AE" | "BE";
    min_percentage: number;
    max_percentage: number;
    default_percentage_mapping: number | null;
}

export interface ScaleProfileWithRanges extends ScaleProfile {
    ranges: ScaleRange[];
}

export interface AssessmentSession {
    id: string;
    class_id: string;
    learning_area_id: string;
    academic_term_id: string;
    academic_year_id: string;
    name: string;
    evaluation_method: "QUANTITATIVE" | "RUBRIC";
    max_points: number | null;
    grading_scale_profile_id: string | null;
    status: "DRAFT" | "PENDING_APPROVAL" | "PUBLISHED";
    rejection_comment: string | null;
    submitted_by: string | null;
    approved_by: string | null;
    scheduled_date: string | null;
    created_at: string;
    updated_at: string;
}

export interface StudentScore {
    id: string;
    session_id: string;
    student_id: string;
    raw_score: number | null;
    calculated_percentage: number | null;
    final_performance_level: "EE" | "ME" | "AE" | "BE" | null;
    enrollment_status: string;
}

export interface OutcomeGrade {
    id: string;
    session_id: string;
    student_id: string;
    performance_indicator_id: string;
    awarded_level: "EE" | "ME" | "AE" | "BE";
}

export interface ParentAssessmentView {
    session_id: string;
    session_name: string;
    evaluation_method: "QUANTITATIVE" | "RUBRIC";
    scheduled_date: string | null;
    raw_score: number | null;
    max_points: number | null;
    performance_level: "EE" | "ME" | "AE" | "BE" | null;
    outcome_grades?: OutcomeGrade[];
}

export interface StudentTermGrade {
    learning_area_id: string;
    learning_area_name: string;
    learning_area_code: string;
    final_level: "EE" | "ME" | "AE" | "BE";
    assessment_count: number;
}

// ── List response shapes ──────────────────────────────────────────────────

export interface ScaleProfileListResult {
    items: ScaleProfile[];
    total: number;
    page: number;
    limit: number;
}

export interface ScaleRangesListResult {
    items: ScaleRange[];
}

export interface SessionListResult {
    items: AssessmentSession[];
    total: number;
    page: number;
    limit: number;
}

export interface StudentScoresResult {
    items: StudentScore[];
}

export interface OutcomeGradesResult {
    items: OutcomeGrade[];
}

export interface ParentAssessmentsResult {
    items: ParentAssessmentView[];
}

export interface StudentTermGradesResult {
    items: StudentTermGrade[];
}

// ── Request payload shapes ────────────────────────────────────────────────

export interface CreateScaleProfilePayload {
    name: string;
    ranges?: ScaleRangePayload[];
}

export interface ScaleRangePayload {
    performance_level: "EE" | "ME" | "AE" | "BE";
    min_percentage: number;
    max_percentage: number;
    default_percentage_mapping?: number | null;
}

export interface BulkSetRangesPayload {
    ranges: ScaleRangePayload[];
}

export interface CreateSessionPayload {
    class_id: string;
    learning_area_id: string;
    academic_term_id: string;
    academic_year_id: string;
    name: string;
    evaluation_method: "QUANTITATIVE" | "RUBRIC";
    max_points?: number | null;
    grading_scale_profile_id?: string | null;
    scheduled_date?: string | null;
}

export interface StudentScorePayload {
    student_id: string;
    raw_score?: number | null;
}

export interface OutcomeGradePayload {
    student_id: string;
    performance_indicator_id: string;
    awarded_level: "EE" | "ME" | "AE" | "BE";
}

export interface BulkUpsertScoresPayload {
    scores: StudentScorePayload[];
}

export interface BulkUpsertOutcomeGradesPayload {
    grades: OutcomeGradePayload[];
}

export interface RejectSessionPayload {
    rejection_comment: string;
}

// ── List params ───────────────────────────────────────────────────────────

export interface ListSessionsParams {
    class_id?: string;
    learning_area_id?: string;
    academic_term_id?: string;
    status?: string;
    evaluation_method?: string;
    search?: string;
    page?: number;
    limit?: number;
    filters?: Record<string, string[]>;
}

// ════════════════════════════════════════════════════════════════════════════
// GRADING SCALE PROFILES
// ════════════════════════════════════════════════════════════════════════════

/** Create a new grading scale profile. SCHOOL_ADMIN only. */
export async function createScaleProfile(
    payload: CreateScaleProfilePayload
): Promise<{ id: string }> {
    return api.post("/api/v1/grading/profiles", payload);
}

/** List all scale profiles for the active school. */
export async function listScaleProfiles(activeOnly = false): Promise<ScaleProfileListResult> {
    const params = activeOnly ? "?active_only=true" : "";
    return api.get(`/api/v1/grading/profiles${params}`);
}

/** Get a single scale profile. Set includeRanges=true to include nested ranges. */
export async function getScaleProfile(
    id: string,
    includeRanges = false
): Promise<ScaleProfile | ScaleProfileWithRanges> {
    const params = includeRanges ? "?include_ranges=true" : "";
    return api.get(`/api/v1/grading/profiles/${id}${params}`);
}

/** Toggle a profile's is_active flag. SCHOOL_ADMIN only. */
export async function toggleScaleProfileActive(
    id: string,
    isActive: boolean
): Promise<{ message: string }> {
    return api.put(`/api/v1/grading/profiles/${id}/toggle?is_active=${isActive}`);
}

/** Delete a scale profile (fails with 409 if sessions reference it). SCHOOL_ADMIN only. */
export async function deleteScaleProfile(id: string): Promise<{ message: string }> {
    return api.delete(`/api/v1/grading/profiles/${id}`);
}

// ════════════════════════════════════════════════════════════════════════════
// GRADING SCALE RANGES
// ════════════════════════════════════════════════════════════════════════════

/** Replace all ranges for a profile. SCHOOL_ADMIN only. */
export async function bulkSetScaleRanges(
    profileId: string,
    payload: BulkSetRangesPayload
): Promise<{ ids: string[] }> {
    return api.put(`/api/v1/grading/profiles/${profileId}/ranges`, payload);
}

/** Get all ranges for a profile. */
export async function getScaleRanges(profileId: string): Promise<ScaleRangesListResult> {
    return api.get(`/api/v1/grading/profiles/${profileId}/ranges`);
}

/** Delete a single range from a profile. SCHOOL_ADMIN only. */
export async function deleteScaleRange(
    profileId: string,
    rangeId: string
): Promise<{ message: string }> {
    return api.delete(`/api/v1/grading/profiles/${profileId}/ranges/${rangeId}`);
}

// ════════════════════════════════════════════════════════════════════════════
// ASSESSMENT SESSIONS
// ════════════════════════════════════════════════════════════════════════════

/** Create a new assessment session (starts in DRAFT). */
export async function createSession(payload: CreateSessionPayload): Promise<{ id: string }> {
    return api.post("/api/v1/assessments/sessions", payload);
}

/** List assessment sessions with optional filters. */
export async function listSessions(params: ListSessionsParams = {}): Promise<SessionListResult> {
    const searchParams = new URLSearchParams();

    // Multi-value filters from DataTable (e.g. status)
    const statuses = params.filters?.status ?? [];
    for (const s of statuses) {
        searchParams.append("status", s);
    }
    const evalMethods = params.filters?.evaluation_method ?? [];
    for (const em of evalMethods) {
        searchParams.append("evaluation_method", em);
    }

    if (params.class_id) searchParams.set("class_id", params.class_id);
    if (params.learning_area_id) searchParams.set("learning_area_id", params.learning_area_id);
    if (params.academic_term_id) searchParams.set("academic_term_id", params.academic_term_id);
    if (params.status) searchParams.set("status", params.status);
    if (params.evaluation_method) searchParams.set("evaluation_method", params.evaluation_method);
    if (params.search) searchParams.set("search", params.search);
    if (params.page) searchParams.set("page", String(params.page));
    if (params.limit) searchParams.set("limit", String(params.limit));

    const qs = searchParams.toString();
    return api.get(`/api/v1/assessments/sessions${qs ? `?${qs}` : ""}`);
}

/** Get a single assessment session by ID. */
export async function getSession(id: string): Promise<AssessmentSession> {
    return api.get(`/api/v1/assessments/sessions/${id}`);
}

// ── Workflow ──────────────────────────────────────────────────────────────

/** Submit a session for admin approval (DRAFT → PENDING_APPROVAL). */
export async function submitSession(id: string): Promise<{ message: string }> {
    return api.post(`/api/v1/assessments/sessions/${id}/submit`);
}

/** Approve and publish a session (PENDING_APPROVAL → PUBLISHED). SCHOOL_ADMIN only. */
export async function approveSession(id: string): Promise<{ message: string }> {
    return api.post(`/api/v1/assessments/sessions/${id}/approve`);
}

/** Reject a session back to draft (PENDING_APPROVAL → DRAFT). SCHOOL_ADMIN only. */
export async function rejectSession(
    id: string,
    payload: RejectSessionPayload
): Promise<{ message: string }> {
    return api.post(`/api/v1/assessments/sessions/${id}/reject`, payload);
}

// ════════════════════════════════════════════════════════════════════════════
// STUDENT SCORES (Quantitative)
// ════════════════════════════════════════════════════════════════════════════

/** Bulk-upsert quantitative raw scores for a session. */
export async function bulkUpsertScores(
    sessionId: string,
    payload: BulkUpsertScoresPayload
): Promise<{ message: string }> {
    return api.post(`/api/v1/assessments/sessions/${sessionId}/scores`, payload);
}

/** Get all quantitative scores for a session. */
export async function getStudentScores(sessionId: string): Promise<StudentScoresResult> {
    return api.get(`/api/v1/assessments/sessions/${sessionId}/scores`);
}

// ════════════════════════════════════════════════════════════════════════════
// STUDENT OUTCOME GRADES (Rubric)
// ════════════════════════════════════════════════════════════════════════════

/** Bulk-upsert rubric outcome grades for a session. */
export async function bulkUpsertOutcomeGrades(
    sessionId: string,
    payload: BulkUpsertOutcomeGradesPayload
): Promise<{ message: string }> {
    return api.post(`/api/v1/assessments/sessions/${sessionId}/grades`, payload);
}

/** Get all outcome grades for a session. */
export async function getOutcomeGrades(sessionId: string): Promise<OutcomeGradesResult> {
    return api.get(`/api/v1/assessments/sessions/${sessionId}/grades`);
}

// ════════════════════════════════════════════════════════════════════════════
// PARENT VIEW & REPORT CARDS
// ════════════════════════════════════════════════════════════════════════════

/** Get all published assessments for a student in a term. */
export async function getParentAssessments(
    studentId: string,
    academicTermId: string
): Promise<ParentAssessmentsResult> {
    return api.get(
        `/api/v1/parent/students/${studentId}/assessments?academic_term_id=${academicTermId}`
    );
}

/** Compile term report card using the "Last One" chronological mode aggregator. */
export async function getStudentTermGrades(
    studentId: string,
    academicTermId: string
): Promise<StudentTermGradesResult> {
    return api.get(
        `/api/v1/parent/students/${studentId}/report-card?academic_term_id=${academicTermId}`
    );
}

// ════════════════════════════════════════════════════════════════════════════
// ASSESSMENT WEIGHT CONFIGS
// ════════════════════════════════════════════════════════════════════════════

/** A KNEC weight configuration entry. */
export interface AssessmentWeightConfig {
    id: string;
    grade_level: string;
    assessment_type_code: string;
    target_exam: string;
    weight_percent: number;
    effective_from: number;
    notes: string | null;
    created_at: string;
}

/** Payload for creating a weight config. */
export interface CreateWeightConfigPayload {
    grade_level: string;
    assessment_type_code: string;
    target_exam: string;
    weight_percent: number;
    effective_from: number;
    notes?: string | null;
}

/** List result for weight configs. */
export interface WeightConfigListResult {
    items: AssessmentWeightConfig[];
    total: number;
    page: number;
    limit: number;
}

/** List weight configs with optional filters. */
export async function listWeightConfigs(params?: {
    grade_level?: string;
    target_exam?: string;
    effective_from?: number;
}): Promise<WeightConfigListResult> {
    const searchParams = new URLSearchParams();
    if (params?.grade_level) searchParams.set("grade_level", params.grade_level);
    if (params?.target_exam) searchParams.set("target_exam", params.target_exam);
    if (params?.effective_from) searchParams.set("effective_from", String(params.effective_from));
    const qs = searchParams.toString();
    return api.get(`/api/v1/assessments/weight-configs${qs ? `?${qs}` : ""}`);
}

/** Get a single weight config by ID. */
export async function getWeightConfig(id: string): Promise<AssessmentWeightConfig> {
    return api.get(`/api/v1/assessments/weight-configs/${id}`);
}

/** Create a new weight config. SCHOOL_ADMIN only. */
export async function createWeightConfig(
    payload: CreateWeightConfigPayload
): Promise<{ id: string }> {
    return api.post("/api/v1/assessments/weight-configs", payload);
}
