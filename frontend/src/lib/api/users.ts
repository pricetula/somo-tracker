/**
 * Users API functions.
 *
 * Endpoints:
 *   GET /api/v1/users — list active staff users by role
 */

import { api } from "./client";

// ─── Types ─────────────────────────────────────────────────────────────────

export interface User {
    id: string;
    email: string;
    first_name: string;
    last_name: string;
    phone_number?: string;
    role: string;
    is_active: boolean;
    created_at: string;
}

export interface ListUsersParams {
    role: "SCHOOL_ADMIN" | "NURSE" | "FINANCE";
    page?: number;
    limit?: number;
}

export interface ListUsersResponse {
    users: User[];
    total: number;
}

// ─── API Functions ─────────────────────────────────────────────────────────

/**
 * List active staff users by role.
 * tenant_id and school_id are resolved server-side from the session.
 */
export async function listUsers(params: ListUsersParams): Promise<ListUsersResponse> {
    const searchParams = new URLSearchParams();
    searchParams.set("role", params.role);
    if (params.page) searchParams.set("page", String(params.page));
    if (params.limit) searchParams.set("limit", String(params.limit));

    const qs = searchParams.toString();
    return api.get<ListUsersResponse>(`/api/v1/users?${qs}`);
}
