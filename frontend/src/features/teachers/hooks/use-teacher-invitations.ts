/**
 * Teacher Invitations feature — mutation hooks for bulk member invitations.
 */

"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
    createTeacherInvitations,
    getTeacherInvitationJob,
    retryTeacherInvitations,
} from "@/lib/api/teacher-invitations";
import { getErrorMessage } from "@/lib/errors";
import type { InvitationRow } from "@/lib/api/teacher-invitations";
import { toast } from "sonner";

export const teacherInvitationKeys = {
    bulk: ["teacher", "invitation", "bulk"] as const,
    job: (jobId: string) => ["teacher", "invitation", "job", jobId] as const,
    retry: (jobId: string) => ["teacher", "invitation", "retry", jobId] as const,
};

export interface BulkInvitePayload {
    invitations: { email: string; full_name: string }[];
    idempotencyKey?: string;
}

export function useBulkInviteTeachers() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationKey: teacherInvitationKeys.bulk,
        mutationFn: (payload: { invitations: InvitationRow[]; idempotencyKey?: string }) => {
            const idempotencyKey = payload.idempotencyKey ?? crypto.randomUUID();
            return createTeacherInvitations({ invitations: payload.invitations }, idempotencyKey);
        },
        onSuccess: (data) => {
            toast.success(data.message ?? "Invitation job queued");
            queryClient.invalidateQueries({ queryKey: ["teachers"] });
            if (data?.job_id) {
                queryClient.invalidateQueries({ queryKey: teacherInvitationKeys.job(data.job_id) });
            }
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

export function useTeacherInvitationJob(jobId: string | undefined) {
    return useQuery({
        queryKey: teacherInvitationKeys.job(jobId ?? ""),
        queryFn: async () => {
            if (!jobId) throw new Error("jobId required");
            return getTeacherInvitationJob(jobId);
        },
        enabled: !!jobId,
    });
}

export function useRetryTeacherInvitations() {
    return useMutation({
        mutationKey: teacherInvitationKeys.retry("retry"),
        mutationFn: (jobId: string) => retryTeacherInvitations(jobId),
        onSuccess: (data) => {
            toast.success(data.message ?? `Retry queued`);
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}
