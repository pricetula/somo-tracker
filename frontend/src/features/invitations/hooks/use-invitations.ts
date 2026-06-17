/**
 * TanStack Query hooks for invitations.
 */

"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import {
    listInvitations,
    createInvitations,
    type ListInvitationsParams,
    type ListInvitationsResponse,
    type CreateInvitationsRequest,
    type CreateInvitationsResponse,
} from "@/lib/api/invitations";
import { getApiErrorMessage } from "@/lib/api/auth";

// ─── Query keys ───────────────────────────────────────────────────────────

export const invitationKeys = {
    all: ["invitations"] as const,
    list: (filters: ListInvitationsParams) => ["invitations", "list", filters] as const,
};

// ─── Hooks ─────────────────────────────────────────────────────────────────

/** Fetch invitations with optional filters. */
export function useInvitations(opts: ListInvitationsParams & { enabled?: boolean } = {}) {
    const {
        search = "",
        email = "",
        status,
        role,
        expired,
        page = 1,
        per_page = 50,
        enabled = true,
    } = opts;

    const filters: ListInvitationsParams = { search, email, status, role, expired, page, per_page };

    return useQuery<ListInvitationsResponse>({
        queryKey: invitationKeys.list(filters),
        queryFn: () => listInvitations(filters),
        placeholderData: (prev) => prev,
        enabled,
    });
}

/** Create new invitations. Invalidates invitation list on success. */
export function useCreateInvitations() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: CreateInvitationsRequest) => createInvitations(payload),
        onSuccess: (result: CreateInvitationsResponse) => {
            queryClient.invalidateQueries({ queryKey: invitationKeys.all });
            if (result.sent > 0) {
                toast.success("Invitations sent", {
                    description: `${result.sent} invitation${result.sent !== 1 ? "s" : ""} sent successfully.`,
                });
            }
            if (result.failed > 0 && result.errors) {
                const errors = result.errors
                    .slice(0, 3)
                    .map((e) => `${e.email}: ${e.error}`)
                    .join(", ");
                toast.error(`${result.failed} invitation${result.failed !== 1 ? "s" : ""} failed`, {
                    description: errors,
                });
            }
        },
        onError: (err) => {
            toast.error("Failed to send invitations", {
                description: getApiErrorMessage(err),
            });
        },
    });
}
