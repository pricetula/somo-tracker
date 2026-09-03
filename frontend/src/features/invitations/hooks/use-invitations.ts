/**
 * useInvitations — TanStack Query hooks for invitation operations.
 */

"use client";

import React from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";

import { getInvitationCount } from "@/lib/api/invitations";
import { STALE_TIMES } from "@/lib/query-config";

// ─── Query keys ───────────────────────────────────────────────────────────

export const invitationKeys = {
    all: ["invitations"] as const,
    count: (role: string) => [...invitationKeys.all, "count", role] as const,
};

// ─── Hooks ────────────────────────────────────────────────────────────────

/** Fetch the pending invitation count for a given role. */
export function useInvitationCount(role: string) {
    return useQuery({
        queryKey: invitationKeys.count(role),
        queryFn: () => getInvitationCount(role),
        staleTime: STALE_TIMES.REFERENCE_DATA,
    });
}

export function useInvalidateInvitationCount(role: string) {
    const queryClient = useQueryClient();

    return React.useCallback(() => {
        queryClient.invalidateQueries({ queryKey: invitationKeys.count(role) });
    }, [queryClient, role]);
}
