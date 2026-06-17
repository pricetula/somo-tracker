/**
 * TanStack Query hooks for members (teachers and staff).
 */

"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import {
    listMembers,
    bulkInvite,
    type ListMembersResponse,
    type BulkInviteRequest,
    type BulkInviteResponse,
} from "@/lib/api/members";
import { getApiErrorMessage } from "@/lib/api/auth";

// ─── Query keys ───────────────────────────────────────────────────────────

export const memberKeys = {
    all: ["members"] as const,
    list: (role: string, filters: { page?: number; per_page?: number; search?: string }) =>
        ["members", role, "list", filters] as const,
};

// ─── Hooks ─────────────────────────────────────────────────────────────────

/** Fetch members by role with pagination and search. */
export function useMembers(
    role: "TEACHER" | "SUPPORT_STAFF",
    opts: { page?: number; per_page?: number; search?: string; enabled?: boolean } = {}
) {
    const { page = 1, per_page = 50, search = "", enabled = true } = opts;

    return useQuery<ListMembersResponse>({
        queryKey: memberKeys.list(role, { page, per_page, search }),
        queryFn: () => listMembers(role, { page, per_page, search }),
        placeholderData: (prev) => prev,
        enabled,
    });
}

/** Bulk invite new members. Invalidates member list on success. */
export function useBulkInvite() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: BulkInviteRequest) => bulkInvite(payload),
        onSuccess: (result: BulkInviteResponse) => {
            queryClient.invalidateQueries({ queryKey: memberKeys.all });
            if (result.sent > 0) {
                toast.success("Invitations sent", {
                    description: `${result.sent} invitation${result.sent !== 1 ? "s" : ""} sent successfully.`,
                });
            }
            if (result.failed > 0 && result.errors) {
                // Show the first few errors
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
