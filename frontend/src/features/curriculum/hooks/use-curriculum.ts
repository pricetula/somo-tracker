/**
 * TanStack Query hooks for the Curriculum feature.
 *
 * Covers learning areas, strands, sub-strands, and performance indicators.
 */

"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import {
    listLearningAreas,
    createLearningArea,
    updateLearningArea,
    deleteLearningArea,
    getLearningAreaTree,
    createStrand,
    updateStrand,
    deleteStrand,
    createSubStrand,
    updateSubStrand,
    deleteSubStrand,
    createPerformanceIndicator,
    updatePerformanceIndicator,
    deletePerformanceIndicator,
    type ListLearningAreasResponse,
    type LearningAreaTree,
    type CreateLearningAreaPayload,
    type UpdateLearningAreaPayload,
    type CreateStrandPayload,
    type UpdateStrandPayload,
    type CreateSubStrandPayload,
    type UpdateSubStrandPayload,
    type CreatePerformanceIndicatorPayload,
    type UpdatePerformanceIndicatorPayload,
} from "@/lib/api/curriculum";
import { getErrorMessage } from "@/lib/errors";

// ─── Query keys ───────────────────────────────────────────────────────────

export const curriculumKeys = {
    all: ["curriculum"] as const,
    learningAreas: {
        all: () => [...curriculumKeys.all, "learning-areas"] as const,
        list: (params?: Record<string, unknown>) =>
            [...curriculumKeys.learningAreas.all(), "list", params] as const,
        detail: (id: string) => [...curriculumKeys.learningAreas.all(), "detail", id] as const,
        tree: (id: string) => [...curriculumKeys.learningAreas.all(), "tree", id] as const,
    },
    strands: {
        all: () => [...curriculumKeys.all, "strands"] as const,
        list: (learningAreaId: string) =>
            [...curriculumKeys.strands.all(), "list", learningAreaId] as const,
    },
    subStrands: {
        all: () => [...curriculumKeys.all, "sub-strands"] as const,
        list: (strandId: string) => [...curriculumKeys.subStrands.all(), "list", strandId] as const,
    },
    performanceIndicators: {
        all: () => [...curriculumKeys.all, "performance-indicators"] as const,
        list: (subStrandId: string) =>
            [...curriculumKeys.performanceIndicators.all(), "list", subStrandId] as const,
    },
};

// ─── Hooks: Learning Areas ────────────────────────────────────────────────

/** Fetch learning areas list, optionally filtered by education_level. */
export function useLearningAreas(
    params: { education_level?: string } = {},
    opts: { enabled?: boolean } = {}
) {
    const { education_level } = params;
    const { enabled = true } = opts;

    return useQuery<ListLearningAreasResponse>({
        queryKey: curriculumKeys.learningAreas.list({ education_level }),
        queryFn: () => listLearningAreas({ education_level }),
        placeholderData: (prev) => prev,
        enabled,
    });
}

/** Fetch a single learning area's full tree (strands → sub-strands → indicators). */
export function useLearningAreaTree(id: string, opts: { enabled?: boolean } = {}) {
    const { enabled = true } = opts;

    return useQuery<LearningAreaTree>({
        queryKey: curriculumKeys.learningAreas.tree(id),
        queryFn: () => getLearningAreaTree(id),
        enabled: enabled && !!id,
    });
}

/** Create a learning area. */
export function useCreateLearningArea() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: CreateLearningAreaPayload) => createLearningArea(data),
        onSuccess: () => {
            queryClient.invalidateQueries({
                queryKey: curriculumKeys.learningAreas.all(),
            });
            toast.success("Learning area created");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

/** Update a learning area. */
export function useUpdateLearningArea() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, data }: { id: string; data: UpdateLearningAreaPayload }) =>
            updateLearningArea(id, data),
        onSuccess: () => {
            queryClient.invalidateQueries({
                queryKey: curriculumKeys.learningAreas.all(),
            });
            toast.success("Learning area updated");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

/** Delete a learning area. */
export function useDeleteLearningArea() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => deleteLearningArea(id),
        onSuccess: () => {
            queryClient.invalidateQueries({
                queryKey: curriculumKeys.learningAreas.all(),
            });
            toast.success("Learning area deleted");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

// ─── Hooks: Strands ───────────────────────────────────────────────────────

/** Create a strand. Invalidates the parent learning area tree. */
export function useCreateStrand() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: CreateStrandPayload) => createStrand(data),
        onSuccess: (_, variables) => {
            queryClient.invalidateQueries({
                queryKey: curriculumKeys.learningAreas.tree(variables.learning_area_id),
            });
            toast.success("Strand created");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

/** Update a strand. */
export function useUpdateStrand() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({
            id,
            data,
            learningAreaId,
        }: {
            id: string;
            data: UpdateStrandPayload;
            learningAreaId: string;
        }) => updateStrand(id, data).then(() => ({ learningAreaId })),
        onSuccess: (result) => {
            queryClient.invalidateQueries({
                queryKey: curriculumKeys.learningAreas.tree(result.learningAreaId),
            });
            toast.success("Strand updated");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

/** Delete a strand. */
export function useDeleteStrand() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, learningAreaId }: { id: string; learningAreaId: string }) =>
            deleteStrand(id).then(() => ({ learningAreaId })),
        onSuccess: (result) => {
            queryClient.invalidateQueries({
                queryKey: curriculumKeys.learningAreas.tree(result.learningAreaId),
            });
            toast.success("Strand deleted");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

// ─── Hooks: Sub-Strands ───────────────────────────────────────────────────

/** Create a sub-strand. Invalidates the parent learning area tree. */
export function useCreateSubStrand() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: CreateSubStrandPayload) => createSubStrand(data),
        onSuccess: () => {
            queryClient.invalidateQueries({
                queryKey: curriculumKeys.learningAreas.all(),
            });
            toast.success("Sub-strand created");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

/** Update a sub-strand. */
export function useUpdateSubStrand() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({
            id,
            data,
            learningAreaId,
        }: {
            id: string;
            data: UpdateSubStrandPayload;
            learningAreaId: string;
        }) => updateSubStrand(id, data).then(() => ({ learningAreaId })),
        onSuccess: (result) => {
            queryClient.invalidateQueries({
                queryKey: curriculumKeys.learningAreas.tree(result.learningAreaId),
            });
            toast.success("Sub-strand updated");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

/** Delete a sub-strand. */
export function useDeleteSubStrand() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, learningAreaId }: { id: string; learningAreaId: string }) =>
            deleteSubStrand(id).then(() => ({ learningAreaId })),
        onSuccess: (result) => {
            queryClient.invalidateQueries({
                queryKey: curriculumKeys.learningAreas.tree(result.learningAreaId),
            });
            toast.success("Sub-strand deleted");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

// ─── Hooks: Performance Indicators ────────────────────────────────────────

/** Create a performance indicator. Invalidates the parent tree. */
export function useCreatePerformanceIndicator() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: CreatePerformanceIndicatorPayload) => createPerformanceIndicator(data),
        onSuccess: () => {
            queryClient.invalidateQueries({
                queryKey: curriculumKeys.learningAreas.all(),
            });
            toast.success("Performance indicator created");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

/** Update a performance indicator. */
export function useUpdatePerformanceIndicator() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({
            id,
            data,
            learningAreaId,
        }: {
            id: string;
            data: UpdatePerformanceIndicatorPayload;
            learningAreaId: string;
        }) => updatePerformanceIndicator(id, data).then(() => ({ learningAreaId })),
        onSuccess: (result) => {
            queryClient.invalidateQueries({
                queryKey: curriculumKeys.learningAreas.tree(result.learningAreaId),
            });
            toast.success("Performance indicator updated");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

/** Delete a performance indicator. */
export function useDeletePerformanceIndicator() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, learningAreaId }: { id: string; learningAreaId: string }) =>
            deletePerformanceIndicator(id).then(() => ({ learningAreaId })),
        onSuccess: (result) => {
            queryClient.invalidateQueries({
                queryKey: curriculumKeys.learningAreas.tree(result.learningAreaId),
            });
            toast.success("Performance indicator deleted");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}
