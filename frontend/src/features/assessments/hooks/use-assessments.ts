/**
 * React Query hooks for the assessments feature.
 */

"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import * as api from "@/lib/api/assessments";
import { getErrorMessage } from "@/lib/errors";
import { STALE_TIMES } from "@/lib/query-config";
import type { ScaleProfile } from "@/lib/api/assessments";
import { toast } from "sonner";

// ─── Query keys ───────────────────────────────────────────────────────────

export const assessmentKeys = {
    profiles: {
        all: ["scale-profiles"] as const,
        list: (activeOnly = false) => ["scale-profiles", "list", activeOnly] as const,
        detail: (id: string) => ["scale-profiles", id] as const,
        ranges: (profileId: string) => ["scale-ranges", profileId] as const,
    },
    sessions: {
        all: ["assessment-sessions"] as const,
        list: () => ["assessment-sessions", "list"] as const,
        detail: (id: string) => ["assessment-sessions", id] as const,
        scores: (sessionId: string) => ["assessment-scores", sessionId] as const,
        grades: (sessionId: string) => ["outcome-grades", sessionId] as const,
    },
    parent: {
        assessments: (studentId: string, termId: string) =>
            ["parent-assessments", studentId, termId] as const,
        reportCard: (studentId: string, termId: string) =>
            ["report-card", studentId, termId] as const,
    },
};

// ════════════════════════════════════════════════════════════════════════════
// HOOKS — Grading Scale Profiles
// ════════════════════════════════════════════════════════════════════════════

/** Fetch all scale profiles for the active school. */
export function useScaleProfileList(activeOnly = false) {
    return useQuery({
        queryKey: assessmentKeys.profiles.list(activeOnly),
        queryFn: () => api.listScaleProfiles(activeOnly),
        staleTime: STALE_TIMES.STANDARD,
    });
}

/** Fetch a single scale profile (optionally with ranges). */
export function useScaleProfile(id: string, includeRanges = false) {
    return useQuery({
        queryKey: includeRanges
            ? [...assessmentKeys.profiles.detail(id), "with-ranges"]
            : assessmentKeys.profiles.detail(id),
        queryFn: () => api.getScaleProfile(id, includeRanges) as Promise<ScaleProfile>,
        enabled: !!id,
        staleTime: STALE_TIMES.STANDARD,
    });
}

/** Fetch ranges for a profile. */
export function useScaleRanges(profileId: string) {
    return useQuery({
        queryKey: assessmentKeys.profiles.ranges(profileId),
        queryFn: () => api.getScaleRanges(profileId),
        enabled: !!profileId,
        staleTime: 2 * 60 * 1000,
    });
}

/** Create a new scale profile with optimistic update. */
export function useCreateScaleProfile() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: (payload: api.CreateScaleProfilePayload) => api.createScaleProfile(payload),
        onMutate: async () => {
            await queryClient.cancelQueries({ queryKey: assessmentKeys.profiles.all });
            const previousQueries = queryClient.getQueriesData({
                queryKey: assessmentKeys.profiles.all,
            });
            return { previousQueries };
        },
        onError: (err, _payload, context) => {
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }
            toast.error(getErrorMessage(err));
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: assessmentKeys.profiles.all });
        },
    });
}

/** Toggle a profile's is_active flag with optimistic update. */
export function useToggleScaleProfile() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: ({ id, isActive }: { id: string; isActive: boolean }) =>
            api.toggleScaleProfileActive(id, isActive),
        onMutate: async ({ id, isActive }) => {
            await queryClient.cancelQueries({ queryKey: assessmentKeys.profiles.all });
            const previousQueries = queryClient.getQueriesData<api.ScaleProfileListResult>({
                queryKey: assessmentKeys.profiles.all,
            });

            queryClient.setQueriesData<api.ScaleProfileListResult>(
                { queryKey: assessmentKeys.profiles.all },
                (old) => {
                    if (!old) return old;
                    return {
                        ...old,
                        items: old.items.map((p) =>
                            p.id === id ? { ...p, is_active: isActive } : p
                        ),
                    };
                }
            );

            return { previousQueries };
        },
        onError: (err, _vars, context) => {
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }
            toast.error(getErrorMessage(err));
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: assessmentKeys.profiles.all });
        },
    });
}

/** Delete a scale profile with optimistic removal. */
export function useDeleteScaleProfile() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => api.deleteScaleProfile(id),
        onMutate: async (id) => {
            await queryClient.cancelQueries({ queryKey: assessmentKeys.profiles.all });
            const previousQueries = queryClient.getQueriesData<{ items: { id: string }[] }>({
                queryKey: assessmentKeys.profiles.all,
            });

            queryClient.setQueriesData<{ items: { id: string }[] }>(
                { queryKey: assessmentKeys.profiles.all },
                (old) => {
                    if (!old) return old;
                    return {
                        ...old,
                        items: old.items.filter((item) => item.id !== id),
                    };
                }
            );

            return { previousQueries };
        },
        onError: (err, _id, context) => {
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }
            toast.error(getErrorMessage(err));
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: assessmentKeys.profiles.all });
        },
    });
}

/** Bulk-set ranges for a profile with optimistic update. */
export function useBulkSetScaleRanges() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: ({
            profileId,
            payload,
        }: {
            profileId: string;
            payload: api.BulkSetRangesPayload;
        }) => api.bulkSetScaleRanges(profileId, payload),
        onMutate: async ({ profileId }) => {
            await queryClient.cancelQueries({
                queryKey: assessmentKeys.profiles.ranges(profileId),
            });
            const previousQueries = queryClient.getQueriesData({
                queryKey: assessmentKeys.profiles.ranges(profileId),
            });
            return { previousQueries };
        },
        onError: (err, _vars, context) => {
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }
            toast.error(getErrorMessage(err));
        },
        onSettled: (_data, _err, variables) => {
            queryClient.invalidateQueries({
                queryKey: assessmentKeys.profiles.ranges(variables.profileId),
            });
            queryClient.invalidateQueries({
                queryKey: assessmentKeys.profiles.detail(variables.profileId),
            });
        },
    });
}

// ════════════════════════════════════════════════════════════════════════════
// HOOKS — Assessment Sessions
// ════════════════════════════════════════════════════════════════════════════

/** Fetch grading data (roster + existing scores/grades) for a session. */
export function useGradingData(sessionId: string) {
    return useQuery({
        queryKey: ["grading-data", sessionId],
        queryFn: () => api.getGradingData(sessionId),
        enabled: !!sessionId,
        staleTime: STALE_TIMES.FREQUENT,
    });
}

/** Fetch paginated assessment sessions. */
export function useSessionList() {
    return useQuery({
        queryKey: assessmentKeys.sessions.list(),
        queryFn: () => api.listSessions(),
        staleTime: STALE_TIMES.FREQUENT,
    });
}

/** Fetch a single assessment session. */
export function useSession(id: string) {
    return useQuery({
        queryKey: assessmentKeys.sessions.detail(id),
        queryFn: () => api.getSession(id),
        enabled: !!id,
        staleTime: STALE_TIMES.FREQUENT,
    });
}

/** Fetch quantitative scores for a session. */
export function useStudentScores(sessionId: string) {
    return useQuery({
        queryKey: assessmentKeys.sessions.scores(sessionId),
        queryFn: () => api.getStudentScores(sessionId),
        enabled: !!sessionId,
        staleTime: STALE_TIMES.FREQUENT,
    });
}

/** Fetch rubric outcome grades for a session. */
export function useOutcomeGrades(sessionId: string) {
    return useQuery({
        queryKey: assessmentKeys.sessions.grades(sessionId),
        queryFn: () => api.getOutcomeGrades(sessionId),
        enabled: !!sessionId,
        staleTime: STALE_TIMES.FREQUENT,
    });
}

/** Create a new assessment session with optimistic update. */
export function useCreateSession() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: (payload: api.CreateSessionPayload) => api.createSession(payload),
        onMutate: async () => {
            await queryClient.cancelQueries({ queryKey: assessmentKeys.sessions.all });
            const previousQueries = queryClient.getQueriesData({
                queryKey: assessmentKeys.sessions.all,
            });
            return { previousQueries };
        },
        onError: (err, _payload, context) => {
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }
            toast.error(getErrorMessage(err));
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: assessmentKeys.sessions.all });
        },
    });
}

/** Submit a session for approval (DRAFT → PENDING_APPROVAL) with optimistic update. */
export function useSubmitSession() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => api.submitSession(id),
        onMutate: async (id) => {
            await queryClient.cancelQueries({ queryKey: assessmentKeys.sessions.all });
            const previousQueries = queryClient.getQueriesData<{
                items: { id: string; status?: string }[];
            }>({
                queryKey: assessmentKeys.sessions.all,
            });

            queryClient.setQueriesData<{ items: { id: string; status?: string }[] }>(
                { queryKey: assessmentKeys.sessions.all },
                (old) => {
                    if (!old) return old;
                    return {
                        ...old,
                        items: old.items.map((s) =>
                            s.id === id ? { ...s, status: "PENDING_APPROVAL" } : s
                        ),
                    };
                }
            );

            return { previousQueries };
        },
        onError: (err, _id, context) => {
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }
            toast.error(getErrorMessage(err));
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: assessmentKeys.sessions.all });
        },
    });
}

/** Approve and publish a session with optimistic update. SCHOOL_ADMIN only. */
export function useApproveSession() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => api.approveSession(id),
        onMutate: async (id) => {
            await queryClient.cancelQueries({ queryKey: assessmentKeys.sessions.all });
            const previousQueries = queryClient.getQueriesData<{
                items: { id: string; status?: string }[];
            }>({
                queryKey: assessmentKeys.sessions.all,
            });

            queryClient.setQueriesData<{ items: { id: string; status?: string }[] }>(
                { queryKey: assessmentKeys.sessions.all },
                (old) => {
                    if (!old) return old;
                    return {
                        ...old,
                        items: old.items.map((s) =>
                            s.id === id ? { ...s, status: "APPROVED" } : s
                        ),
                    };
                }
            );

            return { previousQueries };
        },
        onError: (err, _id, context) => {
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }
            toast.error(getErrorMessage(err));
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: assessmentKeys.sessions.all });
        },
    });
}

/** Delete a DRAFT session permanently with optimistic removal. */
export function useDeleteSession() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => api.deleteSession(id),
        onMutate: async (id) => {
            await queryClient.cancelQueries({ queryKey: assessmentKeys.sessions.all });
            const previousQueries = queryClient.getQueriesData<{ items: { id: string }[] }>({
                queryKey: assessmentKeys.sessions.all,
            });

            queryClient.setQueriesData<{ items: { id: string }[] }>(
                { queryKey: assessmentKeys.sessions.all },
                (old) => {
                    if (!old) return old;
                    return {
                        ...old,
                        items: old.items.filter((item) => item.id !== id),
                    };
                }
            );

            return { previousQueries };
        },
        onError: (err, _id, context) => {
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }
            toast.error(getErrorMessage(err));
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: assessmentKeys.sessions.all });
        },
    });
}

/** Reject a session back to draft with optimistic update. SCHOOL_ADMIN only. */
export function useRejectSession() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: ({ id, rejection_comment }: { id: string; rejection_comment: string }) =>
            api.rejectSession(id, { rejection_comment }),
        onMutate: async ({ id }) => {
            await queryClient.cancelQueries({ queryKey: assessmentKeys.sessions.all });
            const previousQueries = queryClient.getQueriesData<{
                items: { id: string; status?: string }[];
            }>({
                queryKey: assessmentKeys.sessions.all,
            });

            queryClient.setQueriesData<{ items: { id: string; status?: string }[] }>(
                { queryKey: assessmentKeys.sessions.all },
                (old) => {
                    if (!old) return old;
                    return {
                        ...old,
                        items: old.items.map((s) => (s.id === id ? { ...s, status: "DRAFT" } : s)),
                    };
                }
            );

            return { previousQueries };
        },
        onError: (err, _vars, context) => {
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }
            toast.error(getErrorMessage(err));
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: assessmentKeys.sessions.all });
        },
    });
}

/** Bulk-upsert quantitative scores with optimistic update. */
export function useBulkUpsertScores() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: ({
            sessionId,
            payload,
        }: {
            sessionId: string;
            payload: api.BulkUpsertScoresPayload;
        }) => api.bulkUpsertScores(sessionId, payload),
        onMutate: async ({ sessionId }) => {
            await queryClient.cancelQueries({
                queryKey: assessmentKeys.sessions.scores(sessionId),
            });
            const previousQueries = queryClient.getQueriesData({
                queryKey: assessmentKeys.sessions.scores(sessionId),
            });
            return { previousQueries };
        },
        onError: (err, _vars, context) => {
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }
            toast.error(getErrorMessage(err));
        },
        onSettled: (_data, _err, variables) => {
            queryClient.invalidateQueries({
                queryKey: assessmentKeys.sessions.scores(variables.sessionId),
            });
        },
    });
}

/** Bulk-upsert rubric outcome grades with optimistic update. */
export function useBulkUpsertOutcomeGrades() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: ({
            sessionId,
            payload,
        }: {
            sessionId: string;
            payload: api.BulkUpsertOutcomeGradesPayload;
        }) => api.bulkUpsertOutcomeGrades(sessionId, payload),
        onMutate: async ({ sessionId }) => {
            await queryClient.cancelQueries({
                queryKey: assessmentKeys.sessions.grades(sessionId),
            });
            const previousQueries = queryClient.getQueriesData({
                queryKey: assessmentKeys.sessions.grades(sessionId),
            });
            return { previousQueries };
        },
        onError: (err, _vars, context) => {
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }
            toast.error(getErrorMessage(err));
        },
        onSettled: (_data, _err, variables) => {
            queryClient.invalidateQueries({
                queryKey: assessmentKeys.sessions.grades(variables.sessionId),
            });
        },
    });
}

// ════════════════════════════════════════════════════════════════════════════
// HOOKS — Parent View
// ════════════════════════════════════════════════════════════════════════════

/** Fetch published assessments for a student in a term. */
export function useParentAssessments(studentId: string, termId: string) {
    return useQuery({
        queryKey: assessmentKeys.parent.assessments(studentId, termId),
        queryFn: () => api.getParentAssessments(studentId, termId),
        enabled: !!studentId && !!termId,
    });
}

/** Fetch compiled term report card for a student. */
export function useStudentTermGrades(studentId: string, termId: string) {
    return useQuery({
        queryKey: assessmentKeys.parent.reportCard(studentId, termId),
        queryFn: () => api.getStudentTermGrades(studentId, termId),
        enabled: !!studentId && !!termId,
    });
}

// ════════════════════════════════════════════════════════════════════════════
// HOOKS — Weight Configs
// ════════════════════════════════════════════════════════════════════════════

/** Fetch weight configs with optional filters. */
export function useWeightConfigList(params?: {
    grade_level?: string;
    target_exam?: string;
    effective_from?: number;
}) {
    return useQuery({
        queryKey: ["weight-configs", params],
        queryFn: () => api.listWeightConfigs(params),
        staleTime: 5 * 60 * 1000,
    });
}

/** Delete a weight config with optimistic removal. SYSTEM_ADMIN only. */
export function useDeleteWeightConfig() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => api.deleteWeightConfig(id),
        onMutate: async (id) => {
            await queryClient.cancelQueries({ queryKey: ["weight-configs"] });
            const previousQueries = queryClient.getQueriesData<{ items: { id: string }[] }>({
                queryKey: ["weight-configs"],
            });

            queryClient.setQueriesData<{ items: { id: string }[] }>(
                { queryKey: ["weight-configs"] },
                (old) => {
                    if (!old) return old;
                    return {
                        ...old,
                        items: old.items.filter((item) => item.id !== id),
                    };
                }
            );

            return { previousQueries };
        },
        onError: (err, _id, context) => {
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }
            toast.error(getErrorMessage(err));
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: ["weight-configs"] });
        },
    });
}

/** Create a new weight config with optimistic update. */
export function useCreateWeightConfig() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: (payload: api.CreateWeightConfigPayload) => api.createWeightConfig(payload),
        onMutate: async () => {
            await queryClient.cancelQueries({ queryKey: ["weight-configs"] });
            const previousQueries = queryClient.getQueriesData({
                queryKey: ["weight-configs"],
            });
            return { previousQueries };
        },
        onError: (err, _payload, context) => {
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }
            toast.error(getErrorMessage(err));
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: ["weight-configs"] });
        },
    });
}
