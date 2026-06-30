/**
 * Invitations API functions.
 *
 * Endpoints:
 *   GET  /api/v1/invitations — list invitations with filters
 */

import { api } from "./client";
import type {
    Invitation,
    InvitationStatus,
    InvitationRole,
    ListInvitationsResponse,
} from "./generated";

// ─── Re-export generated types ───────────────────────────────────────────

export type { Invitation, InvitationStatus, InvitationRole, ListInvitationsResponse };

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
