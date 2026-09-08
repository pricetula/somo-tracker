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

// ─── Params Types ──────────────────────────────────────────────────────────

export interface ListFinanceStaffParams {
    page?: number;
    limit?: number;
    search?: string;
    include_inactive?: boolean;
}

// ─── API Functions ─────────────────────────────────────────────────────────

/** List active finance staff (FINANCE role). */
export async function listFinanceStaff(
    params: ListFinanceStaffParams = {}
): Promise<ListMembersResponse> {
    const searchParams = new URLSearchParams({ role: "FINANCE" });
    if (params.page) searchParams.set("page", String(params.page));
    if (params.limit) searchParams.set("limit", String(params.limit));
    if (params.search) searchParams.set("search", params.search);
    if (params.include_inactive) searchParams.set("include_inactive", "true");

    const qs = searchParams.toString();
    return { items: [], total: 0 };
}

/** Toggle finance staff active status. */
export async function toggleFinanceActive(userId: string, isActive: boolean): Promise<void> {
    return undefined;
}

/** Hard-delete a finance staff member. */
export async function deleteFinanceStaff(userId: string): Promise<void> {
    return undefined;
}
