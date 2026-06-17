/**
 * Invitations API functions.
 *
 * Endpoints:
 *   GET  /api/v1/invitations — list invitations with filters
 *   POST /api/v1/invitations — create new invitations
 */

import { api } from "./client";

// ─── Types ─────────────────────────────────────────────────────────────────

export type InvitationStatus = "pending" | "accepted" | "expired" | "revoked";

export type InvitationRole = "SYSTEM_ADMIN" | "SCHOOL_ADMIN" | "TEACHER" | "SUPPORT_STAFF";

export interface Invitation {
    id: string;
    school_id: string;
    tenant_id: string;
    email: string;
    role: InvitationRole;
    status: InvitationStatus;
    first_name?: string;
    last_name?: string;
    invited_by?: string;
    expires_at: string;
    accepted_at?: string;
    created_at: string;
}

export interface ListInvitationsParams {
    search?: string;
    email?: string;
    status?: InvitationStatus;
    role?: InvitationRole;
    expired?: boolean;
    page?: number;
    per_page?: number;
}

export interface ListInvitationsResponse {
    invitations: Invitation[];
    total: number;
}

export interface CreateInvitationItem {
    email: string;
    first_name?: string;
    last_name?: string;
    role: InvitationRole;
}

export interface CreateInvitationsRequest {
    invites: CreateInvitationItem[];
}

export interface CreateInvitationsResponse {
    sent: number;
    failed: number;
    errors?: Array<{ email: string; error: string }>;
}

// ─── API Functions ─────────────────────────────────────────────────────────

/** List invitations with optional filters. */
export async function listInvitations(
    params: ListInvitationsParams = {}
): Promise<ListInvitationsResponse> {
    const searchParams = new URLSearchParams();
    if (params.search) searchParams.set("search", params.search);
    if (params.email) searchParams.set("email", params.email);
    if (params.status) searchParams.set("status", params.status);
    if (params.role) searchParams.set("role", params.role);
    if (params.expired !== undefined) searchParams.set("expired", String(params.expired));
    if (params.page) searchParams.set("page", String(params.page));
    if (params.per_page) searchParams.set("per_page", String(params.per_page));

    const qs = searchParams.toString();
    return api.get<ListInvitationsResponse>(`/api/v1/invitations${qs ? `?${qs}` : ""}`);
}

/** Create new invitations (bulk). */
export async function createInvitations(
    payload: CreateInvitationsRequest
): Promise<CreateInvitationsResponse> {
    return api.post<CreateInvitationsResponse>("/api/v1/invitations", payload);
}
