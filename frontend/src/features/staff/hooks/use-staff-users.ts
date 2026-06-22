/**
 * TanStack Query hooks for listing active staff users and invitations.
 *
 * Two independent query hooks:
 *   useStaffUsers   — GET /api/v1/users?role=...
 *   useStaffInvitations — GET /api/v1/invitations?role=...&status[]=...
 *
 * Each manages its own loading, error, and empty states independently.
 */

"use client";

import { useQuery } from "@tanstack/react-query";

import { listUsers, type ListUsersResponse, type ListUsersParams } from "@/lib/api/users";
import {
    listInvitationsByRole,
    type ListInvitationsResponse,
    type InvitationStatus,
} from "@/lib/api/invitations";

// ─── Query keys ───────────────────────────────────────────────────────────

export const staffKeys = {
    all: ["staff"] as const,
    users: (role: string) => ["staff", "users", role] as const,
    invitations: (role: string) => ["staff", "invitations", role] as const,
};

// ─── Hooks ─────────────────────────────────────────────────────────────────

/** Fetch active staff users by role. */
export function useStaffUsers(
    role: ListUsersParams["role"],
    opts: { page?: number; limit?: number; enabled?: boolean } = {}
) {
    const { page = 1, limit = 50, enabled = true } = opts;

    return useQuery<ListUsersResponse>({
        queryKey: [...staffKeys.users(role), { page, limit }],
        queryFn: () => listUsers({ role, page, limit }),
        placeholderData: (prev) => prev,
        enabled,
    });
}

/** Fetch non-accepted invitations by role, filtered by explicit status array. */
export function useStaffInvitations(
    role: "SCHOOL_ADMIN" | "NURSE" | "FINANCE",
    opts: { page?: number; limit?: number; enabled?: boolean } = {}
) {
    const { page = 1, limit = 50, enabled = true } = opts;

    // Never include 'accepted' — those users appear in the users list
    const status: InvitationStatus[] = ["pending", "expired", "revoked", "invite_failed"];

    return useQuery<ListInvitationsResponse>({
        queryKey: [...staffKeys.invitations(role), { page, limit, status }],
        queryFn: () => listInvitationsByRole({ role, status, page, limit }),
        placeholderData: (prev) => prev,
        enabled,
    });
}
