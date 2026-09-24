import { api } from "./client";

export interface TeacherListItem {
    membership_id: string;
    user_id: string;
    email: string;
    full_name: string;
    invited_at?: string | null;
    accepted_at?: string | null;
    is_active: boolean;
    created_at: string;
}

export interface TeacherListResponse {
    items: TeacherListItem[];
    total: number;
    page: number;
    limit: number;
}

export interface ListTeachersParams {
    page?: number;
    limit?: number;
    search?: string;
    invitation_status?: "invited" | "accepted" | "all";
}

export async function listTeachers(params: ListTeachersParams = {}): Promise<TeacherListResponse> {
    const qs = new URLSearchParams();
    if (params.page) qs.set("page", String(params.page));
    if (params.limit) qs.set("limit", String(params.limit));
    if (params.search) qs.set("search", params.search);
    if (params.invitation_status) qs.set("invitation_status", params.invitation_status);

    const url = `/api/teachers${qs.toString() ? `?${qs}` : ""}`;
    return api.get<TeacherListResponse>(url);
}

export interface DeleteTeachersResponse {
    code: string;
    message: string;
    errors: Record<string, unknown>;
}

export async function deleteTeachers(userIds: string[]): Promise<DeleteTeachersResponse> {
    return api.delete<DeleteTeachersResponse>("/api/teachers", { user_ids: userIds });
}

export type TeacherSummary = {
    total_teachers: number;
    teachers_without_assignment: number;
};

export async function getTeacherSummary(): Promise<TeacherSummary> {
    return api.get<TeacherSummary>("/api/teachers/summary");
}
