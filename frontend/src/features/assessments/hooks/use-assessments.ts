/**
 * React Query hooks for the assessments feature.
 */

"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import * as api from "@/lib/api/assessments";
import { getErrorMessage } from "@/lib/errors";
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
        staleTime: 2 * 60 * 1000,
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
        staleTime: 2 * 60 * 1000,
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

/** Create a new scale profile. */
export function useCreateScaleProfile() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: (payload: api.CreateScaleProfilePayload) => api.createScaleProfile(payload),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: assessmentKeys.profiles.all });
            toast.success("Scale profile created.");
        },
        onError: (err) => toast.error(getErrorMessage(err)),
    });
}

/** Toggle a profile's is_active flag. */
export function useToggleScaleProfile() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: ({ id, isActive }: { id: string; isActive: boolean }) =>
            api.toggleScaleProfileActive(id, isActive),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: assessmentKeys.profiles.all });
            toast.success("Profile updated.");
        },
        onError: (err) => toast.error(getErrorMessage(err)),
    });
}

/** Delete a scale profile. */
export function useDeleteScaleProfile() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => api.deleteScaleProfile(id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: assessmentKeys.profiles.all });
            toast.success("Profile deleted.");
        },
        onError: (err) => toast.error(getErrorMessage(err)),
    });
}

/** Bulk-set ranges for a profile. */
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
        onSuccess: (_data, variables) => {
            queryClient.invalidateQueries({
                queryKey: assessmentKeys.profiles.ranges(variables.profileId),
            });
            queryClient.invalidateQueries({
                queryKey: assessmentKeys.profiles.detail(variables.profileId),
            });
            toast.success("Ranges saved.");
        },
        onError: (err) => toast.error(getErrorMessage(err)),
    });
}

// ════════════════════════════════════════════════════════════════════════════
// HOOKS — Assessment Sessions
// ════════════════════════════════════════════════════════════════════════════

/** Fetch paginated assessment sessions. */
export function useSessionList() {
    return useQuery({
        queryKey: assessmentKeys.sessions.list(),
        queryFn: () => api.listSessions(),
        staleTime: 30_000,
    });
}

/** Fetch a single assessment session. */
export function useSession(id: string) {
    return useQuery({
        queryKey: assessmentKeys.sessions.detail(id),
        queryFn: () => api.getSession(id),
        enabled: !!id,
        staleTime: 30_000,
    });
}

/** Fetch quantitative scores for a session. */
export function useStudentScores(sessionId: string) {
    return useQuery({
        queryKey: assessmentKeys.sessions.scores(sessionId),
        queryFn: () => api.getStudentScores(sessionId),
        enabled: !!sessionId,
        staleTime: 30_000,
    });
}

/** Fetch rubric outcome grades for a session. */
export function useOutcomeGrades(sessionId: string) {
    return useQuery({
        queryKey: assessmentKeys.sessions.grades(sessionId),
        queryFn: () => api.getOutcomeGrades(sessionId),
        enabled: !!sessionId,
        staleTime: 30_000,
    });
}

/** Create a new assessment session. */
export function useCreateSession() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: (payload: api.CreateSessionPayload) => api.createSession(payload),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: assessmentKeys.sessions.all });
            toast.success("Assessment session created.");
        },
        onError: (err) => toast.error(getErrorMessage(err)),
    });
}

/** Submit a session for approval (DRAFT → PENDING_APPROVAL). */
export function useSubmitSession() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => api.submitSession(id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: assessmentKeys.sessions.all });
            toast.success("Session submitted for approval.");
        },
        onError: (err) => toast.error(getErrorMessage(err)),
    });
}

/** Approve and publish a session. SCHOOL_ADMIN only. */
export function useApproveSession() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => api.approveSession(id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: assessmentKeys.sessions.all });
            toast.success("Session approved and published.");
        },
        onError: (err) => toast.error(getErrorMessage(err)),
    });
}

/** Reject a session back to draft. SCHOOL_ADMIN only. */
export function useRejectSession() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: ({ id, rejection_comment }: { id: string; rejection_comment: string }) =>
            api.rejectSession(id, { rejection_comment }),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: assessmentKeys.sessions.all });
            toast.success("Session rejected and returned to draft.");
        },
        onError: (err) => toast.error(getErrorMessage(err)),
    });
}

/** Bulk-upsert quantitative scores. */
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
        onSuccess: (_data, variables) => {
            queryClient.invalidateQueries({
                queryKey: assessmentKeys.sessions.scores(variables.sessionId),
            });
            toast.success("Scores saved.");
        },
        onError: (err) => toast.error(getErrorMessage(err)),
    });
}

/** Bulk-upsert rubric outcome grades. */
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
        onSuccess: (_data, variables) => {
            queryClient.invalidateQueries({
                queryKey: assessmentKeys.sessions.grades(variables.sessionId),
            });
            toast.success("Grades saved.");
        },
        onError: (err) => toast.error(getErrorMessage(err)),
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
