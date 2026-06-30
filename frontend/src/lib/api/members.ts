/**
 * Members API functions.
 *
 * Endpoints:
 *   GET  /api/v1/members?role=...  — list members by role
 */

import { api } from "./client";
import type { Member, ListMembersResponse } from "./generated";

// ─── Re-export generated types ───────────────────────────────────────────

export type { Member, ListMembersResponse };

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
