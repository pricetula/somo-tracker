import { api } from "./client";

export interface FinanceListItem {
    membership_id: string;
    user_id: string;
    email: string;
    full_name: string;
    invited_at?: string | null;
    accepted_at?: string | null;
    is_active: boolean;
    created_at: string;
}

export interface FinanceListResponse {
    items: FinanceListItem[];
    total: number;
    page: number;
    limit: number;
}

export interface ListFinanceParams {
    page?: number;
    limit?: number;
    search?: string;
    invitation_status?: "invited" | "accepted" | "all";
}

export async function listFinance(params: ListFinanceParams = {}): Promise<FinanceListResponse> {
    const qs = new URLSearchParams();
    if (params.page) qs.set("page", String(params.page));
    if (params.limit) qs.set("limit", String(params.limit));
    if (params.search) qs.set("search", params.search);
    if (params.invitation_status) qs.set("invitation_status", params.invitation_status);

    const url = `/api/finance${qs.toString() ? `?${qs}` : ""}`;
    return api.get<FinanceListResponse>(url);
}

export interface DeleteFinanceResponse {
    code: string;
    message: string;
    errors: Record<string, unknown>;
}

export async function deleteFinance(userIds: string[]): Promise<DeleteFinanceResponse> {
    return api.delete<DeleteFinanceResponse>("/api/finance", { user_ids: userIds });
}
