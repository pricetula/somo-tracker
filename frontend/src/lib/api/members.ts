/**
 * Members API functions.
 *
 * Endpoints:
 *   GET  /api/v1/members?role=...  — list members by role
 *   POST /api/v1/members/invite    — bulk invite new members
 */

import { api } from "./client";

// ─── Types ─────────────────────────────────────────────────────────────────

export interface Member {
    id: string;
    email: string;
    first_name: string;
    last_name: string;
    role: "TEACHER" | "SUPPORT_STAFF";
    is_active: boolean;
    created_at: string;
}

export interface ListMembersResponse {
    members: Member[];
    total: number;
}

export interface BulkInviteRequest {
    role: "TEACHER" | "SUPPORT_STAFF";
    invites: InviteItem[];
}

export interface InviteItem {
    email: string;
    first_name: string;
    last_name: string;
}

export interface BulkInviteResponse {
    sent: number;
    failed: number;
    errors?: InviteErrorItem[];
}

export interface InviteErrorItem {
    email: string;
    error: string;
}

// ─── API Functions ─────────────────────────────────────────────────────────

/** List members by role with pagination and optional search. */
export async function listMembers(
    role: "TEACHER" | "SUPPORT_STAFF",
    params: { page?: number; per_page?: number; search?: string } = {}
): Promise<ListMembersResponse> {
    const searchParams = new URLSearchParams({ role });
    if (params.page) searchParams.set("page", String(params.page));
    if (params.per_page) searchParams.set("per_page", String(params.per_page));
    if (params.search) searchParams.set("search", params.search);

    const qs = searchParams.toString();
    return api.get<ListMembersResponse>(`/api/v1/members?${qs}`);
}

/** Bulk invite new members with a given role. */
export async function bulkInvite(payload: BulkInviteRequest): Promise<BulkInviteResponse> {
    return api.post<BulkInviteResponse>("/api/v1/members/invite", payload);
}
