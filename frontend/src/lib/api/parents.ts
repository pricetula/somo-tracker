/**
 * Parents API functions.
 *
 * Endpoints (from backend/internal/parents/handler.go):
 *   POST   /api/v1/parents                       — create parent
 *   GET    /api/v1/parents                       — list parents
 *   GET    /api/v1/parents/:id                   — get parent detail
 *   PUT    /api/v1/parents/:id                   — update parent
 *   DELETE /api/v1/parents/:id                   — delete parent
 *   POST   /api/v1/parents/:parent_id/students   — link student
 *   DELETE /api/v1/parents/:parent_id/students/:student_id — unlink student
 */

import { api } from "./client";

// ─── Domain Types ─────────────────────────────────────────────────────────

export interface Parent {
    id: string;
    user_id: string;
    full_name: string;
    email: string;
    phone_number: string;
    is_active: boolean;
    created_at: string;
}

export interface StudentLink {
    student_id: string;
    full_name: string;
    relationship?: string | null;
    is_primary: boolean;
}

export interface ParentDetail {
    id: string;
    user_id: string;
    full_name: string;
    email: string;
    phone_number: string;
    is_active: boolean;
    created_at: string;
    linked_students: StudentLink[];
}

// ─── Response Types ───────────────────────────────────────────────────────

export interface ListParentsResponse {
    items: Parent[];
    total: number;
    page: number;
    limit: number;
}

export interface ParentDetailResponse {
    data: ParentDetail;
}

export interface CreateParentResponse {
    id: string;
}

// ─── Payload Types ────────────────────────────────────────────────────────

export interface CreateParentPayload {
    email: string;
    full_name: string;
    phone_number: string;
}

export interface UpdateParentPayload {
    phone_number?: string;
    is_active?: boolean;
}

export interface LinkStudentPayload {
    student_id: string;
    relationship?: string | null;
    is_primary?: boolean;
}

// ─── Params Types ──────────────────────────────────────────────────────────

export interface ListParentsParams {
    search?: string;
    student_id?: string;
    page?: number;
    limit?: number;
    /** Filter values keyed by FilterItem id, e.g. { education_level: ["Early_Years"], grade_level: ["G1", "G2"] } */
    filters?: Record<string, string[]>;
}

// ─── API Functions ─────────────────────────────────────────────────────────

/** List parents, optionally filtered by search, student_id, or curriculum filters (education_level, grade_level), with pagination. */
export async function listParents(params: ListParentsParams = {}): Promise<ListParentsResponse> {
    const searchParams = new URLSearchParams();

    // Multi-value curriculum filters
    const edLevels = params.filters?.education_level ?? [];
    for (const el of edLevels) {
        searchParams.append("education_level", el);
    }
    const grLevels = params.filters?.grade_level ?? [];
    for (const gl of grLevels) {
        searchParams.append("grade_level", gl);
    }

    if (params.search) searchParams.set("search", params.search);
    if (params.student_id) searchParams.set("student_id", params.student_id);
    if (params.page) searchParams.set("page", String(params.page));
    if (params.limit) searchParams.set("limit", String(params.limit));

    const qs = searchParams.toString();
    return api.get<ListParentsResponse>(`/api/v1/parents?${qs}`);
}

/** Create a new parent profile. */
export async function createParent(data: CreateParentPayload): Promise<CreateParentResponse> {
    return api.post<CreateParentResponse>("/api/v1/parents", data);
}

/** Get parent detail with linked students. */
export async function getParentDetail(id: string): Promise<ParentDetailResponse> {
    return api.get<ParentDetailResponse>(`/api/v1/parents/${id}`);
}

/** Get the authenticated parent's own profile with linked children. */
export async function getMyParentProfile(): Promise<ParentDetailResponse> {
    return api.get<ParentDetailResponse>("/api/v1/parents/me");
}

/** Update a parent profile (phone_number, is_active). */
export async function updateParent(id: string, data: UpdateParentPayload): Promise<void> {
    return api.put<void>(`/api/v1/parents/${id}`, data);
}

/** Delete a parent profile. */
export async function deleteParent(id: string): Promise<void> {
    return api.delete<void>(`/api/v1/parents`, { id });
}

/** Link a student to a parent. */
export async function linkStudent(parentId: string, data: LinkStudentPayload): Promise<void> {
    return api.post<void>(`/api/v1/parents/${parentId}/students`, data);
}

/** Unlink a student from a parent. */
export async function unlinkStudent(parentId: string, studentId: string): Promise<void> {
    return api.delete<void>(`/api/v1/parents/student-link`, {
        parent_id: parentId,
        student_id: studentId,
    });
}

// ============================================================================
// Bulk Invite (Import system)
// ============================================================================

import type { ImportResponse } from "./imports";
import { ApiError } from "./client";

export interface InviteRow {
    email: string;
    full_name?: string;
}

export interface BulkParentInviteRequest {
    rows: InviteRow[];
}

/**
 * POST /api/v1/parents/invite — submit a bulk parent invitation job.
 * Accepts an array of email rows. Returns immediately with a job_id for
 * progress polling. Processing happens asynchronously via the Asynq import engine.
 * Compatible with the BulkInviteForm's submitFn interface.
 */
export async function submitParentBulkInvite(body: {
    role: string;
    rows: Array<{ email: string; full_name?: string }>;
}): Promise<ImportResponse> {
    // The role field is accepted for compatibility with BulkInviteForm's submitFn
    // interface but is ignored — the backend endpoint always creates PARENT invites.
    return api.post<ImportResponse>("/api/v1/parents/invite", {
        rows: body.rows,
    });
}

/**
 * Type guard to check if an error is an import_already_in_progress response.
 */
export function getParentImportAlreadyInProgress(err: unknown): string | null {
    if (
        err instanceof ApiError &&
        err.code === "import_already_in_progress" &&
        err.extra?.active_job_id
    ) {
        return String(err.extra.active_job_id);
    }
    return null;
}
