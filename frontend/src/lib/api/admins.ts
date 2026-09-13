import { api } from "./client";

export interface AdminListItem {
    membership_id: string;
    user_id: string;
    email: string;
    full_name: string;
    invited_at?: string | null;
    accepted_at?: string | null;
    is_active: boolean;
    created_at: string;
}

export interface AdminListResponse {
    items: AdminListItem[];
    total: number;
    page: number;
    limit: number;
}

export interface ListAdminsParams {
    page?: number;
    limit?: number;
    search?: string;
    invitation_status?: "invited" | "accepted" | "all";
}

/**
 * List admins with pagination, search and invitation status filter.
 * Returns shape expected by DataTable: { items, total, page, limit }
 */
export async function listAdmins(params: ListAdminsParams = {}): Promise<AdminListResponse> {
    const qs = new URLSearchParams();
    if (params.page) qs.set("page", String(params.page));
    if (params.limit) qs.set("limit", String(params.limit));
    if (params.search) qs.set("search", params.search);
    if (params.invitation_status) qs.set("invitation_status", params.invitation_status);

    const url = `/api/admins${qs.toString() ? `?${qs}` : ""}`;
    return api.get<AdminListResponse>(url);
}
