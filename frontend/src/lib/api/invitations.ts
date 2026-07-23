/**
 * Invitations API functions.
 *
 * Endpoints:
 *   GET   /api/v1/invitations — list invitations with filters
 *   POST  /api/v1/staff/invite — bulk invite staff members
 *   PATCH /api/v1/invitations/:id/revoke — revoke a pending invitation
 */

import { api, ApiError } from "./client";
import type { ImportResponse } from "./imports";
import type {
    Invitation,
    InvitationStatus,
    InvitationRole,
    ListInvitationsResponse,
} from "./generated";

// ─── Re-export generated types ───────────────────────────────────────────

export type { Invitation, InvitationStatus, InvitationRole, ListInvitationsResponse };

/** Response shape for GET /api/v1/invitations/count */
export interface InvitationCountResponse {
    total: number;
}

/**
 * GET /api/v1/invitations/count?role=... — returns total non-expired invitation count for a role.
 */
export async function getInvitationCount(role: string): Promise<InvitationCountResponse> {
    const qs = new URLSearchParams({ role }).toString();
    return api.get<InvitationCountResponse>(`/api/v1/invitations/count?${qs}`);
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

// ============================================================================
// Bulk Invitation Types
// ============================================================================

export interface InviteRow {
    email: string;
    full_name?: string;
}

export interface BulkInviteRequest {
    role: string;
    rows: InviteRow[];
}

/**
 * POST /api/v1/staff/invite — submit a bulk staff invitation job.
 * Accepts a role and an array of email rows. Returns immediately with a
 * job_id for progress polling. Processing happens asynchronously via
 * the same Asynq-based import engine used for student imports.
 */
export async function submitBulkInvite(body: BulkInviteRequest): Promise<ImportResponse> {
    return api.post<ImportResponse>("/api/v1/staff/invite", body);
}

/**
 * PATCH /api/v1/invitations/:id/revoke — revoke a pending invitation.
 * Only SCHOOL_ADMIN can revoke invitations.
 */
export async function revokeInvitation(id: string): Promise<void> {
    return api.patch<void>(`/api/v1/invitations/${id}/revoke`);
}

/**
 * Type guard to check if an error is an import_already_in_progress response.
 * The backend returns this when a CreateJob request is rejected because
 * another job is already active for the same school.
 * Returns the active_job_id if matched, or null otherwise.
 */
export function getImportAlreadyInProgress(err: unknown): string | null {
    if (
        err instanceof ApiError &&
        err.code === "import_already_in_progress" &&
        err.extra?.active_job_id
    ) {
        return String(err.extra.active_job_id);
    }
    return null;
}
