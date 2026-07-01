/**
 * TanStack Query hooks for the Assessment feature.
 *
 * Covers blueprints, sessions, results, and weight configs.
 */

"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import {
    listBlueprints,
    createBlueprint,
    getBlueprintDetail,
    deleteBlueprint,
    linkIndicators,
    unlinkIndicator,
    listSessions,
    createSession,
    getSessionDetail,
    deleteSession,
    batchUpsertResults,
    listResults,
    listWeightConfigs,
} from "@/lib/api/assessment";
import { getErrorMessage } from "@/lib/errors";
import type {
    ListBlueprintsResponse,
    BlueprintDetailResponse,
    CreateBlueprintPayload,
    ListSessionsResponse,
    SessionDetailResponse,
    CreateSessionPayload,
    BatchUpsertResultsPayload,
    ListResultsResponse,
    ListWeightConfigsResponse,
} from "../types";

// ─── Query keys ───────────────────────────────────────────────────────────

export const assessmentKeys = {
    all: ["assessment"] as const,
    blueprints: {
        all: () => [...assessmentKeys.all, "blueprints"] as const,
        list: (params?: Record<string, unknown>) =>
            [...assessmentKeys.blueprints.all(), "list", params] as const,
        detail: (id: string) => [...assessmentKeys.blueprints.all(), "detail", id] as const,
    },
    sessions: {
        all: () => [...assessmentKeys.all, "sessions"] as const,
        list: (params?: Record<string, unknown>) =>
            [...assessmentKeys.sessions.all(), "list", params] as const,
        detail: (id: string) => [...assessmentKeys.sessions.all(), "detail", id] as const,
    },
    results: {
        all: (sessionId: string) =>
            [...assessmentKeys.sessions.detail(sessionId), "results"] as const,
    },
    weightConfigs: {
        all: () => [...assessmentKeys.all, "weight-configs"] as const,
        list: (params?: Record<string, unknown>) =>
            [...assessmentKeys.weightConfigs.all(), "list", params] as const,
    },
};

// ─── Hooks: Blueprints ────────────────────────────────────────────────────

/** Fetch blueprints list, optionally filtered. */
export function useBlueprints(
    params: { grade_level?: string; type?: string; academic_year?: number; term?: number } = {},
    opts: { enabled?: boolean } = {}
) {
    const { enabled = true } = opts;

    return useQuery<ListBlueprintsResponse>({
        queryKey: assessmentKeys.blueprints.list(params),
        queryFn: () => listBlueprints(params),
        placeholderData: (prev) => prev,
        enabled,
    });
}

/** Fetch a single blueprint detail (with linked indicators). */
export function useBlueprintDetail(id: string, opts: { enabled?: boolean } = {}) {
    const { enabled = true } = opts;

    return useQuery<BlueprintDetailResponse>({
        queryKey: assessmentKeys.blueprints.detail(id),
        queryFn: () => getBlueprintDetail(id),
        enabled: enabled && !!id,
    });
}

/** Create a blueprint. */
export function useCreateBlueprint() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: CreateBlueprintPayload) => createBlueprint(data),
        onSuccess: () => {
            queryClient.invalidateQueries({
                queryKey: assessmentKeys.blueprints.all(),
            });
            toast.success("Blueprint created");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

/** Delete a blueprint. */
export function useDeleteBlueprint() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => deleteBlueprint(id),
        onSuccess: () => {
            queryClient.invalidateQueries({
                queryKey: assessmentKeys.blueprints.all(),
            });
            toast.success("Blueprint deleted");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

/** Link indicators to a blueprint. */
export function useLinkIndicators() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({
            blueprintId,
            indicatorIds,
        }: {
            blueprintId: string;
            indicatorIds: string[];
        }) => linkIndicators(blueprintId, indicatorIds),
        onSuccess: (_, variables) => {
            queryClient.invalidateQueries({
                queryKey: assessmentKeys.blueprints.detail(variables.blueprintId),
            });
            toast.success("Indicators linked");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

/** Unlink an indicator from a blueprint. */
export function useUnlinkIndicator() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ blueprintId, indicatorId }: { blueprintId: string; indicatorId: string }) =>
            unlinkIndicator(blueprintId, indicatorId),
        onSuccess: (_, variables) => {
            queryClient.invalidateQueries({
                queryKey: assessmentKeys.blueprints.detail(variables.blueprintId),
            });
            toast.success("Indicator removed");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

// ─── Hooks: Sessions ──────────────────────────────────────────────────────

/** Fetch sessions list, optionally filtered. */
export function useSessions(
    params: { class_id?: string; blueprint_id?: string } = {},
    opts: { enabled?: boolean } = {}
) {
    const { enabled = true } = opts;

    return useQuery<ListSessionsResponse>({
        queryKey: assessmentKeys.sessions.list(params),
        queryFn: () => listSessions(params),
        placeholderData: (prev) => prev,
        enabled,
    });
}

/** Fetch a single session detail (with results). */
export function useSessionDetail(id: string, opts: { enabled?: boolean } = {}) {
    const { enabled = true } = opts;

    return useQuery<SessionDetailResponse>({
        queryKey: assessmentKeys.sessions.detail(id),
        queryFn: () => getSessionDetail(id),
        enabled: enabled && !!id,
    });
}

/** Create a session. */
export function useCreateSession() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: CreateSessionPayload) => createSession(data),
        onSuccess: () => {
            queryClient.invalidateQueries({
                queryKey: assessmentKeys.sessions.all(),
            });
            toast.success("Session created");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

/** Delete a session. */
export function useDeleteSession() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => deleteSession(id),
        onSuccess: () => {
            queryClient.invalidateQueries({
                queryKey: assessmentKeys.sessions.all(),
            });
            toast.success("Session deleted");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

/** Fetch results for a session. */
export function useSessionResults(sessionId: string, opts: { enabled?: boolean } = {}) {
    const { enabled = true } = opts;

    return useQuery<ListResultsResponse>({
        queryKey: assessmentKeys.results.all(sessionId),
        queryFn: () => listResults(sessionId),
        enabled: enabled && !!sessionId,
    });
}

/** Batch upsert rubric results. */
export function useBatchUpsertResults() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ sessionId, data }: { sessionId: string; data: BatchUpsertResultsPayload }) =>
            batchUpsertResults(sessionId, data),
        onSuccess: (_, variables) => {
            queryClient.invalidateQueries({
                queryKey: assessmentKeys.results.all(variables.sessionId),
            });
            queryClient.invalidateQueries({
                queryKey: assessmentKeys.sessions.detail(variables.sessionId),
            });
            toast.success("Scores saved");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

// ─── Hooks: Weight Configs ───────────────────────────────────────────────

/** Fetch weight configs, optionally filtered. */
export function useWeightConfigs(
    params: { grade_level?: string; target_exam?: string } = {},
    opts: { enabled?: boolean } = {}
) {
    const { enabled = true } = opts;

    return useQuery<ListWeightConfigsResponse>({
        queryKey: assessmentKeys.weightConfigs.list(params),
        queryFn: () => listWeightConfigs(params),
        placeholderData: (prev) => prev,
        enabled,
    });
}
