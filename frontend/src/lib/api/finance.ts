/**
 * Finance Staff API functions.
 *
 * Endpoints:
 *   GET  /api/v1/members?role=FINANCE&include_inactive=...
 *   PATCH /api/v1/members/:user_id/active
 */

import { api } from "./client";
import type { Member, ListMembersResponse } from "./generated";

// ─── Re-export generated types ───────────────────────────────────────────

export type { Member, ListMembersResponse };

// ─── API Functions ─────────────────────────────────────────────────────────

/** List active finance staff (FINANCE role). */
export async function listFinanceStaff(
    params: { page?: number; per_page?: number; search?: string; include_inactive?: boolean } = {}
): Promise<ListMembersResponse> {
    const searchParams = new URLSearchParams({ role: "FINANCE" });
    if (params.page) searchParams.set("page", String(params.page));
    if (params.per_page) searchParams.set("per_page", String(params.per_page));
    if (params.search) searchParams.set("search", params.search);
    if (params.include_inactive) searchParams.set("include_inactive", "true");

    const qs = searchParams.toString();
    return api.get<ListMembersResponse>(`/api/v1/members?${qs}`);
}

/** Toggle finance staff active status. */
export async function toggleFinanceActive(userId: string, isActive: boolean): Promise<void> {
    return api.patch<void>(`/api/v1/members/${userId}/active`, { is_active: isActive });
}
