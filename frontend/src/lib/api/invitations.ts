/**
 * Invitations API functions.
 *
 * Endpoints:
 *   GET  /api/v1/invitations — list invitations with filters
 */

import { api } from "./client";

// ─── Types ─────────────────────────────────────────────────────────────────

export type InvitationStatus = "pending" | "accepted" | "expired" | "revoked" | "invite_failed";

export type InvitationRole = "SYSTEM_ADMIN" | "SCHOOL_ADMIN" | "TEACHER" | "NURSE" | "FINANCE";

export interface Invitation {
    id: string;
    school_id: string;
    tenant_id: string;
    email: string;
    role: InvitationRole;
    status: InvitationStatus;
    full_name?: string;
    invited_by?: string;
    expires_at: string;
    accepted_at?: string;
    created_at: string;
}

export interface ListInvitationsResponse {
    invitations: Invitation[];
    total: number;
}

/**
 * List invitations by role, optionally filtered by a single status.
 *
 * Note: The backend does NOT support multi-value status[] params.
 * It accepts only a single `status` string value. To fetch multiple
 * statuses, make separate calls or omit status to get all results.
 */
export async function listInvitationsByRole(params: {
    role: string;
    status?: string;
    page?: number;
    limit?: number;
}): Promise<ListInvitationsResponse> {
    const searchParams = new URLSearchParams();
    searchParams.set("role", params.role);
    if (params.status) searchParams.set("status", params.status);
    if (params.page) searchParams.set("page", String(params.page));
    if (params.limit) searchParams.set("limit", String(params.limit));

    const qs = searchParams.toString();
    return api.get<ListInvitationsResponse>(`/api/v1/invitations?${qs}`);
}
