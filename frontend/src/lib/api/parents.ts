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
    data: Parent[];
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

// ─── API Functions ─────────────────────────────────────────────────────────

/** List parents, optionally filtered by search or student_id. */
export async function listParents(
    params: { search?: string; student_id?: string } = {}
): Promise<ListParentsResponse> {
    const searchParams = new URLSearchParams();
    if (params.search) searchParams.set("search", params.search);
    if (params.student_id) searchParams.set("student_id", params.student_id);

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

/** Update a parent profile (phone_number, is_active). */
export async function updateParent(id: string, data: UpdateParentPayload): Promise<void> {
    return api.put<void>(`/api/v1/parents/${id}`, data);
}

/** Delete a parent profile. */
export async function deleteParent(id: string): Promise<void> {
    return api.delete<void>(`/api/v1/parents/${id}`);
}

/** Link a student to a parent. */
export async function linkStudent(parentId: string, data: LinkStudentPayload): Promise<void> {
    return api.post<void>(`/api/v1/parents/${parentId}/students`, data);
}

/** Unlink a student from a parent. */
export async function unlinkStudent(parentId: string, studentId: string): Promise<void> {
    return api.delete<void>(`/api/v1/parents/${parentId}/students/${studentId}`);
}
