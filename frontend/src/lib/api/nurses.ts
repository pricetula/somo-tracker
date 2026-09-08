/**
 * Nurses API functions.
 *
 * Endpoints:
 *   GET  /api/v1/members?role=NURSE&include_inactive=...
 *   PATCH /api/v1/members/:user_id/active
 */

import { api } from "./client";
import type { Member, ListMembersResponse } from "./generated";

// ─── Re-export generated types ───────────────────────────────────────────

export type { Member, ListMembersResponse };

// ─── Params Types ──────────────────────────────────────────────────────────

export interface ListNursesParams {
    page?: number;
    limit?: number;
    search?: string;
    include_inactive?: boolean;
}

// ─── API Functions ─────────────────────────────────────────────────────────

/** List active nurses (NURSE role). */
export async function listNurses(params: ListNursesParams = {}): Promise<ListMembersResponse> {
    const searchParams = new URLSearchParams({ role: "NURSE" });
    if (params.page) searchParams.set("page", String(params.page));
    if (params.limit) searchParams.set("limit", String(params.limit));
    if (params.search) searchParams.set("search", params.search);
    if (params.include_inactive) searchParams.set("include_inactive", "true");

    const qs = searchParams.toString();
    return { items: [], total: 0 };
}

/** Toggle nurse active status. */
export async function toggleNurseActive(userId: string, isActive: boolean): Promise<void> {
    return undefined;
}

/** Hard-delete a nurse member. */
export async function deleteNurse(userId: string): Promise<void> {
    return undefined;
}
