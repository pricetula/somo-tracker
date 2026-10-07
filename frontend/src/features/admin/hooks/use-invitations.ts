/**
 * Admin Invitations feature — mutation hooks for bulk member invitations.
 */

"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createInvitations, getInvitationJob, retryFailedInvitations } from "@/lib/api/invitations";
import { getErrorMessage } from "@/lib/errors";
import { toast } from "sonner";

interface InvitationRow {
    email: string;
    full_name: string;
}

interface BulkInvitationResponse {
    job_id: string;
    message: string;
    status: string;
    total_records: number;
}

interface InvitationJob {
    id: string;
    status: string;
    message: string;
    total_records: number;
    completed_records: number;
    failed_records: number;
    created_at: string;
    updated_at: string;
}

interface InvitationRetryResponse {
    message: string;
}

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
    const queryClient = useQueryClient();
    return useMutation<BulkInvitationResponse, Error, BulkInvitePayload>({
        mutationKey: invitationKeys.bulk,
        mutationFn: (payload) => {
            const idempotencyKey = payload.idempotencyKey ?? crypto.randomUUID();
            return createInvitations({
                invitations: payload.invitations,
                idempotencyKey,
            });
        },
        onSuccess: (data) => {
            toast.success(data.message ?? "Invitation job queued");
            queryClient.invalidateQueries({ queryKey: ["admins"] });
            if (data.job_id) {
                queryClient.invalidateQueries({ queryKey: invitationKeys.job(data.job_id) });
            }
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
            toast.success(data.message);
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
