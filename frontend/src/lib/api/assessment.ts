/**
 * Assessment API functions.
 *
 * Endpoints (from backend/internal/assessment/handler.go):
 *   Blueprints:  GET|POST /api/v1/assessment/blueprints, GET|PUT|DELETE /:id
 *   Indicators:  POST /:id/indicators, DELETE /:id/indicators/:indicator_id
 *   Sessions:    GET|POST /api/v1/assessment/sessions, GET|PUT|DELETE /:id
 *   Results:     POST /:id/results/batch, GET /:id/results
 *   Weight Conf: GET /api/v1/assessment/weight-configs
 */

import { api } from "./client";
import type {
    CreateBlueprintPayload,
    CreateBlueprintResponse,
    ListBlueprintsResponse,
    BlueprintDetailResponse,
    CreateSessionPayload,
    CreateSessionResponse,
    ListSessionsResponse,
    SessionDetailResponse,
    BatchUpsertResultsPayload,
    ListResultsResponse,
    ListWeightConfigsResponse,
} from "@/features/assessment/types";

// ─── Blueprints ───────────────────────────────────────────────────────────

/** List assessment blueprints for the active school, with optional filters. */
export async function listBlueprints(
    params: { grade_level?: string; type?: string; academic_year?: number; term?: number } = {}
): Promise<ListBlueprintsResponse> {
    const searchParams = new URLSearchParams();
    if (params.grade_level) searchParams.set("grade_level", params.grade_level);
    if (params.type) searchParams.set("type", params.type);
    if (params.academic_year) searchParams.set("academic_year", String(params.academic_year));
    if (params.term) searchParams.set("term", String(params.term));

    const qs = searchParams.toString();
    return api.get<ListBlueprintsResponse>(`/api/v1/assessment/blueprints?${qs}`);
}

/** Create a new assessment blueprint. */
export async function createBlueprint(
    data: CreateBlueprintPayload
): Promise<CreateBlueprintResponse> {
    return api.post<CreateBlueprintResponse>("/api/v1/assessment/blueprints", data);
}

/** Get blueprint detail (metadata + linked indicators). */
export async function getBlueprintDetail(id: string): Promise<BlueprintDetailResponse> {
    return api.get<BlueprintDetailResponse>(`/api/v1/assessment/blueprints/${id}`);
}

/** Update a blueprint (title, type). */
export async function updateBlueprint(
    id: string,
    data: { title?: string; type?: string }
): Promise<void> {
    return api.put<void>(`/api/v1/assessment/blueprints/${id}`, data);
}

/** Delete a blueprint. */
export async function deleteBlueprint(id: string): Promise<void> {
    return api.delete<void>(`/api/v1/assessment/blueprints/${id}`);
}

// ─── Blueprint ↔ Indicator Linking ────────────────────────────────────────

/** Link performance indicators to a blueprint. */
export async function linkIndicators(blueprintId: string, indicatorIds: string[]): Promise<void> {
    return api.post<void>(`/api/v1/assessment/blueprints/${blueprintId}/indicators`, {
        indicator_ids: indicatorIds,
    });
}

/** Unlink a performance indicator from a blueprint. */
export async function unlinkIndicator(blueprintId: string, indicatorId: string): Promise<void> {
    return api.delete<void>(
        `/api/v1/assessment/blueprints/${blueprintId}/indicators/${indicatorId}`
    );
}

// ─── Sessions ─────────────────────────────────────────────────────────────

/** List assessment sessions for the active school, with optional filters. */
export async function listSessions(
    params: { class_id?: string; blueprint_id?: string } = {}
): Promise<ListSessionsResponse> {
    const searchParams = new URLSearchParams();
    if (params.class_id) searchParams.set("class_id", params.class_id);
    if (params.blueprint_id) searchParams.set("blueprint_id", params.blueprint_id);

    const qs = searchParams.toString();
    return api.get<ListSessionsResponse>(`/api/v1/assessment/sessions?${qs}`);
}

/** Create a new assessment session. */
export async function createSession(data: CreateSessionPayload): Promise<CreateSessionResponse> {
    return api.post<CreateSessionResponse>("/api/v1/assessment/sessions", data);
}

/** Get session detail (metadata + rubric results). */
export async function getSessionDetail(id: string): Promise<SessionDetailResponse> {
    return api.get<SessionDetailResponse>(`/api/v1/assessment/sessions/${id}`);
}

/** Update a session (date, upload reference). */
export async function updateSession(
    id: string,
    data: { date_administered?: string; knec_upload_reference?: string | null }
): Promise<void> {
    return api.put<void>(`/api/v1/assessment/sessions/${id}`, data);
}

/** Delete a session. */
export async function deleteSession(id: string): Promise<void> {
    return api.delete<void>(`/api/v1/assessment/sessions/${id}`);
}

// ─── Results ──────────────────────────────────────────────────────────────

/** Batch upsert rubric results for a session. */
export async function batchUpsertResults(
    sessionId: string,
    data: BatchUpsertResultsPayload
): Promise<{ rows_affected: number }> {
    return api.post<{ rows_affected: number }>(
        `/api/v1/assessment/sessions/${sessionId}/results/batch`,
        data
    );
}

/** List all rubric results for a session. */
export async function listResults(sessionId: string): Promise<ListResultsResponse> {
    return api.get<ListResultsResponse>(`/api/v1/assessment/sessions/${sessionId}/results`);
}

// ─── Weight Configs ───────────────────────────────────────────────────────

/** List KNEC weight configs, optionally filtered by grade_level and target_exam. */
export async function listWeightConfigs(
    params: { grade_level?: string; target_exam?: string } = {}
): Promise<ListWeightConfigsResponse> {
    const searchParams = new URLSearchParams();
    if (params.grade_level) searchParams.set("grade_level", params.grade_level);
    if (params.target_exam) searchParams.set("target_exam", params.target_exam);

    const qs = searchParams.toString();
    return api.get<ListWeightConfigsResponse>(`/api/v1/assessment/weight-configs?${qs}`);
}
