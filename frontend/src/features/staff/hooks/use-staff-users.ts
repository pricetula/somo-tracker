/**
 * TanStack Query hooks for listing active staff users and invitations.
 *
 * Two independent query hooks:
 *   useStaffUsers   — GET /api/v1/members?role=...
 *   useStaffInvitations — GET /api/v1/invitations?role=...&status=...
 *
 * Each manages its own loading, error, and empty states independently.
 */

"use client";

import { useQuery } from "@tanstack/react-query";

import { listMembers, type ListMembersResponse } from "@/lib/api/members";
import { listInvitationsByRole, type ListInvitationsResponse } from "@/lib/api/invitations";

// ─── Query keys ───────────────────────────────────────────────────────────

export const staffKeys = {
    all: ["staff"] as const,
    users: (role: string) => ["staff", "users", role] as const,
    invitations: (role: string) => ["staff", "invitations", role] as const,
};

// ─── Hooks ─────────────────────────────────────────────────────────────────

/**
 * Fetch active staff users by role.
 *
 * Maps to GET /api/v1/members?role=... (the actual backend endpoint).
 * Supported roles: NURSE, FINANCE, TEACHER.
 * SCHOOL_ADMIN is NOT supported by the backend members handler.
 */
export function useStaffUsers(
    role: "NURSE" | "FINANCE" | "TEACHER",
    opts: { page?: number; limit?: number; enabled?: boolean } = {}
) {
    const { page = 1, limit = 50, enabled = true } = opts;

    return useQuery<ListMembersResponse>({
        queryKey: [...staffKeys.users(role), { page, limit }],
        queryFn: () => listMembers(role, { page, per_page: limit }),
        placeholderData: (prev) => prev,
        enabled,
    });
}

/**
 * Fetch invitations by role.
 *
 * Note: The backend does NOT support multi-value status[] filtering.
 * It accepts a single `status` string. Omitting status returns all
 * records (pending by default, excluding expired unless `expired=true`).
 */
export function useStaffInvitations(
    role: string,
    opts: { status?: string; page?: number; limit?: number; enabled?: boolean } = {}
) {
    const { status, page = 1, limit = 50, enabled = true } = opts;

    return useQuery<ListInvitationsResponse>({
        queryKey: [...staffKeys.invitations(role), { page, limit, status }],
        queryFn: () => listInvitationsByRole({ role, status, page, limit }),
        placeholderData: (prev) => prev,
        enabled,
    });
}
