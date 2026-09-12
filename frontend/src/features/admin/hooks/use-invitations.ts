/**
 * Admin Invitations feature — mutation hooks for bulk member invitations.
 */

"use client";

import { useMutation, useQuery } from "@tanstack/react-query";
import { createInvitations, getInvitationJob, retryFailedInvitations } from "@/lib/api/invitations";
import type {
    BulkInvitationResponse,
    InvitationJob,
    InvitationRetryResponse,
    InvitationRow,
} from "@/lib/api/generated";
import { getErrorMessage } from "@/lib/errors";
import { toast } from "sonner";

export const invitationKeys = {
    bulk: ["admin", "invitation", "bulk"] as const,
    job: (jobId: string) => ["admin", "invitation", "job", jobId] as const,
    retry: (jobId: string) => ["admin", "invitation", "retry", jobId] as const,
};

export interface BulkInvitePayload {
    invitations: InvitationRow[];
    idempotencyKey?: string;
}

// ─── Mutation: bulk invite ────────────────────────────────────────────────

export function useBulkInviteUsers() {
    return useMutation<BulkInvitationResponse, Error, BulkInvitePayload>({
        mutationKey: invitationKeys.bulk,
        mutationFn: (payload) =>
            createInvitations({
                invitations: payload.invitations,
                idempotencyKey: payload.idempotencyKey,
            }),
        onSuccess: (data) => {
            toast.success(data.message ?? "Invitation job queued");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

// ─── Mutation: retry failed ---

export function useRetryFailedInvitations() {
    return useMutation<InvitationRetryResponse, Error, string>({
        mutationKey: invitationKeys.retry("retry"),
        mutationFn: (jobId) => retryFailedInvitations(jobId),
        onSuccess: (data) => {
            toast.success(data.message ?? `Retry queued (${data.count})`);
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

// ─── Query: invitation job status ---

export function useInvitationJob(jobId: string | undefined) {
    return useQuery<InvitationJob, Error>({
        queryKey: invitationKeys.job(jobId ?? ""),
        queryFn: async () => {
            if (!jobId) throw new Error("jobId required");
            return getInvitationJob(jobId);
        },
        enabled: !!jobId,
    });
}
