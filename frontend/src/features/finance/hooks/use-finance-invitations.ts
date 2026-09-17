/**
 * Finance Invitations feature — mutation hooks for bulk member invitations.
 */

"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
    createFinanceInvitations,
    getFinanceInvitationJob,
    retryFinanceInvitations,
} from "@/lib/api/finance-invitations";
import { getErrorMessage } from "@/lib/errors";
import { toast } from "sonner";

export const financeInvitationKeys = {
    bulk: ["finance", "invitation", "bulk"] as const,
    job: (jobId: string) => ["finance", "invitation", "job", jobId] as const,
    retry: (jobId: string) => ["finance", "invitation", "retry", jobId] as const,
};

export interface BulkInvitePayload {
    invitations: { email: string; full_name: string }[];
    idempotencyKey?: string;
}

export function useBulkInviteFinances() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationKey: financeInvitationKeys.bulk,
        mutationFn: (payload) => {
            const idempotencyKey = payload.idempotencyKey ?? crypto.randomUUID();
            return createFinanceInvitations({ invitations: payload.invitations }, idempotencyKey);
        },
        onSuccess: (data) => {
            toast.success(data.message ?? "Invitation job queued");
            queryClient.invalidateQueries({ queryKey: ["finance"] });
            if (data?.job_id) {
                queryClient.invalidateQueries({ queryKey: financeInvitationKeys.job(data.job_id) });
            }
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

export function useFinanceInvitationJob(jobId: string | undefined) {
    return useQuery({
        queryKey: financeInvitationKeys.job(jobId ?? ""),
        queryFn: async () => {
            if (!jobId) throw new Error("jobId required");
            return getFinanceInvitationJob(jobId);
        },
        enabled: !!jobId,
    });
}

export function useRetryFinanceInvitations() {
    return useMutation({
        mutationKey: financeInvitationKeys.retry("retry"),
        mutationFn: (jobId: string) => retryFinanceInvitations(jobId),
        onSuccess: (data) => {
            toast.success(data.message ?? `Retry queued`);
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}
