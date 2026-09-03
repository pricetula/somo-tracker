/**
 * Assessments API functions.
 *
 * Endpoints:
 *   GET    /api/v1/assessments/sessions                  — list sessions
 *   GET    /api/v1/assessments/sessions/:id             — get session
 *   POST   /api/v1/assessments/sessions                  — create session
 *   PUT    /api/v1/assessments/sessions/:id             — update session
 *   DELETE /api/v1/assessments/sessions/:id             — delete session
 *   POST   /api/v1/assessments/sessions/:id/submit      — submit for approval
 *   POST   /api/v1/assessments/sessions/:id/approve     — approve & publish
 *   POST   /api/v1/assessments/sessions/:id/reject      — reject back to draft
 *   POST   /api/v1/assessments/sessions/:id/scores      — upsert quantitative scores
 *   GET    /api/v1/assessments/sessions/:id/scores      — list scores
 */

import { api } from "./client";

export type EvaluationMethod = "QUANTITATIVE" | "RUBRIC";
export type AssessmentStatus = "DRAFT" | "PENDING_APPROVAL" | "PUBLISHED";

export interface AssessmentSession {
    id: string;
    tenant_id: string;
    school_id: string;
    class_id: string;
    learning_area_id: string;
    academic_term_id: string;
    academic_year_id: string;
    name: string;
    evaluation_method: EvaluationMethod;
    max_points?: number | null;
    grading_scale_profile_id?: string | null;
    status: AssessmentStatus;
    rejection_comment?: string | null;
    submitted_by?: string | null;
    approved_by?: string | null;
    scheduled_date?: string | null;
    created_at: string;
    updated_at: string;
    created_by: string;
}

export interface SessionListResult {
    items: AssessmentSession[];
    total: number;
    page: number;
    limit: number;
}

export interface StudentScore {
    id: string;
    session_id: string;
    student_id: string;
    raw_score: number | null;
    calculated_percentage: number | null;
    final_performance_level: string | null;
    enrollment_status: string;
    created_at: string;
    updated_at: string;
}

export interface ScoreListResult {
    items: StudentScore[];
    total: number;
    page: number;
    limit: number;
}

export interface ListSessionsParams {
    class_id?: string;
    status?: AssessmentStatus;
    evaluation_method?: EvaluationMethod;
    page?: number;
    limit?: number;
}

export async function listSessions(params: ListSessionsParams = {}): Promise<SessionListResult> {
    const sp = new URLSearchParams();
    if (params.class_id) sp.set("class_id", params.class_id);
    if (params.status) sp.set("status", params.status);
    if (params.evaluation_method) sp.set("evaluation_method", params.evaluation_method);
    if (params.page) sp.set("page", String(params.page));
    if (params.limit) sp.set("limit", String(params.limit));
    const qs = sp.toString();
    return api.get<SessionListResult>(`/api/v1/assessments/sessions${qs ? `?${qs}` : ""}`);
}

export async function getSession(id: string): Promise<AssessmentSession> {
    return api.get<AssessmentSession>(`/api/v1/assessments/sessions/${id}`);
}

export async function createSession(payload: {
    class_id: string;
    learning_area_id: string;
    name: string;
    evaluation_method: EvaluationMethod;
    max_points?: number;
    grading_scale_profile_id?: string;
    scheduled_date?: string;
}): Promise<AssessmentSession> {
    return api.post<AssessmentSession>("/api/v1/assessments/sessions", payload);
}

export async function updateSession(
    id: string,
    payload: {
        name: string;
        evaluation_method: EvaluationMethod;
        max_points?: number | null;
        grading_scale_profile_id?: string | null;
        scheduled_date?: string | null;
    }
): Promise<AssessmentSession> {
    return api.put<AssessmentSession>(`/api/v1/assessments/sessions/${id}`, payload);
}

export async function deleteSession(id: string): Promise<void> {
    await api.delete(`/api/v1/assessments/sessions/${id}`);
}

export async function submitSession(id: string): Promise<void> {
    await api.post(`/api/v1/assessments/sessions/${id}/submit`, {});
}

export async function approveSession(id: string): Promise<void> {
    await api.post(`/api/v1/assessments/sessions/${id}/approve`, {});
}

export async function rejectSession(id: string, comment: string): Promise<void> {
    await api.post(`/api/v1/assessments/sessions/${id}/reject`, { comment });
}

export async function upsertScores(
    sessionId: string,
    scores: { student_id: string; raw_score: number | null }[]
): Promise<{ code: string; message: string; count: number }> {
    return api.post(`/api/v1/assessments/sessions/${sessionId}/scores`, { scores });
}

export async function listScores(
    sessionId: string,
    page = 1,
    limit = 50
): Promise<ScoreListResult> {
    return api.get<ScoreListResult>(
        `/api/v1/assessments/sessions/${sessionId}/scores?page=${page}&limit=${limit}`
    );
}

export interface RubricEntry {
    student_id: string;
    performance_indicator_id: string;
    awarded_level: string;
}

export interface RubricOutcome {
    id: string;
    session_id: string;
    student_id: string;
    performance_indicator_id: string;
    awarded_level: string;
}

export async function upsertRubricOutcomes(
    sessionId: string,
    grading: RubricEntry[]
): Promise<{ code: string; message: string; count: number }> {
    return api.post(`/api/v1/assessments/sessions/${sessionId}/rubric-outcomes`, { grading });
}

export async function listRubricOutcomes(sessionId: string): Promise<{ items: RubricOutcome[] }> {
    return api.get<{ items: RubricOutcome[] }>(
        `/api/v1/assessments/sessions/${sessionId}/rubric-outcomes`
    );
}
