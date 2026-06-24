/**
 * Members API functions.
 *
 * Endpoints:
 *   GET  /api/v1/members?role=...  — list members by role
 */

import { api } from "./client";

// ─── Types ─────────────────────────────────────────────────────────────────

export interface Member {
    id: string;
    email: string;
    full_name: string;
    role: "TEACHER" | "NURSE" | "FINANCE";
    is_active: boolean;
    created_at: string;
}

export interface ListMembersResponse {
    members: Member[];
    total: number;
}

// ─── API Functions ─────────────────────────────────────────────────────────

/** List members by role with pagination and optional search. */
export async function listMembers(
    role: "TEACHER" | "NURSE" | "FINANCE",
    params: { page?: number; per_page?: number; search?: string } = {}
): Promise<ListMembersResponse> {
    const searchParams = new URLSearchParams({ role });
    if (params.page) searchParams.set("page", String(params.page));
    if (params.per_page) searchParams.set("per_page", String(params.per_page));
    if (params.search) searchParams.set("search", params.search);

    const qs = searchParams.toString();
    return api.get<ListMembersResponse>(`/api/v1/members?${qs}`);
}
