import { api } from "./client";

export interface GuardianListItem {
    membership_id: string;
    user_id: string;
    email: string;
    full_name: string;
    invited_at?: string | null;
    accepted_at?: string | null;
    is_active: boolean;
    created_at: string;
}

export interface GuardianListResponse {
    items: GuardianListItem[];
    total: number;
    page: number;
    limit: number;
}

export interface ListGuardiansParams {
    page?: number;
    limit?: number;
    search?: string;
    invitation_status?: "invited" | "accepted" | "all";
}

export async function listGuardians(
    params: ListGuardiansParams = {}
): Promise<GuardianListResponse> {
    const qs = new URLSearchParams();
    if (params.page) qs.set("page", String(params.page));
    if (params.limit) qs.set("limit", String(params.limit));
    if (params.search) qs.set("search", params.search);
    if (params.invitation_status) qs.set("invitation_status", params.invitation_status);

    const url = `/api/guardians${qs.toString() ? `?${qs}` : ""}`;
    return api.get<GuardianListResponse>(url);
}

export interface DeleteGuardiansResponse {
    code: string;
    message: string;
    errors: Record<string, unknown>;
}

export async function deleteGuardians(userIds: string[]): Promise<DeleteGuardiansResponse> {
    return api.delete<DeleteGuardiansResponse>("/api/guardians", { user_ids: userIds });
}
