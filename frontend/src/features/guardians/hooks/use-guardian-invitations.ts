/**
 * Guardian Invitations feature — mutation hooks for bulk member invitations.
 */

"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
    createGuardiansInvitations,
    getGuardiansInvitationJob,
    retryGuardiansInvitations,
} from "@/lib/api/guardians-invitations";
import { getErrorMessage } from "@/lib/errors";
import { toast } from "sonner";

export const guardianInvitationKeys = {
    bulk: ["guardian", "invitation", "bulk"] as const,
    job: (jobId: string) => ["guardian", "invitation", "job", jobId] as const,
    retry: (jobId: string) => ["guardian", "invitation", "retry", jobId] as const,
};

export interface BulkInvitePayload {
    invitations: { email: string; full_name: string }[];
    idempotencyKey?: string;
}

export function useBulkInviteGuardians() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationKey: guardianInvitationKeys.bulk,
        mutationFn: (payload) => {
            const idempotencyKey = payload.idempotencyKey ?? crypto.randomUUID();
            return createGuardiansInvitations({ invitations: payload.invitations }, idempotencyKey);
        },
        onSuccess: (data) => {
            toast.success(data.message ?? "Invitation job queued");
            queryClient.invalidateQueries({ queryKey: ["guardians"] });
            if (data?.job_id) {
                queryClient.invalidateQueries({
                    queryKey: guardianInvitationKeys.job(data.job_id),
                });
            }
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

export function useGuardianInvitationJob(jobId: string | undefined) {
    return useQuery({
        queryKey: guardianInvitationKeys.job(jobId ?? ""),
        queryFn: async () => {
            if (!jobId) throw new Error("jobId required");
            return getGuardiansInvitationJob(jobId);
        },
        enabled: !!jobId,
    });
}

export function useRetryGuardianInvitations() {
    return useMutation({
        mutationKey: guardianInvitationKeys.retry("retry"),
        mutationFn: (jobId: string) => retryGuardiansInvitations(jobId),
        onSuccess: (data) => {
            toast.success(data.message ?? `Retry queued`);
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}
