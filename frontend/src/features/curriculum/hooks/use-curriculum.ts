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

export { useSeedDefaultCBC } from "./use-seed-default-cbc";

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

/** Create a learning area with optimistic update. */
export function useCreateLearningArea() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: CreateLearningAreaPayload) => createLearningArea(data),
        onMutate: async () => {
            await queryClient.cancelQueries({
                queryKey: curriculumKeys.learningAreas.all(),
            });
            const previousQueries = queryClient.getQueriesData({
                queryKey: curriculumKeys.learningAreas.all(),
            });
            return { previousQueries };
        },
        onError: (err, _data, context) => {
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }
            toast.error(getErrorMessage(err));
        },
        onSettled: () => {
            queryClient.invalidateQueries({
                queryKey: curriculumKeys.learningAreas.all(),
            });
        },
    });
}

/** Update a learning area with optimistic update. */
export function useUpdateLearningArea() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, data }: { id: string; data: UpdateLearningAreaPayload }) =>
            updateLearningArea(id, data),
        onMutate: async ({ id, data }) => {
            await queryClient.cancelQueries({
                queryKey: curriculumKeys.learningAreas.all(),
            });
            const previousQueries = queryClient.getQueriesData<ListLearningAreasResponse>({
                queryKey: curriculumKeys.learningAreas.all(),
            });

            queryClient.setQueriesData<ListLearningAreasResponse>(
                { queryKey: curriculumKeys.learningAreas.all() },
                (old) => {
                    if (!old) return old;
                    return {
                        ...old,
                        items: old.items.map((la) => (la.id === id ? { ...la, ...data } : la)),
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
            queryClient.invalidateQueries({
                queryKey: curriculumKeys.learningAreas.all(),
            });
        },
    });
}

/** Delete a learning area with optimistic removal. */
export function useDeleteLearningArea() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => deleteLearningArea(id),
        onMutate: async (id) => {
            await queryClient.cancelQueries({
                queryKey: curriculumKeys.learningAreas.all(),
            });
            const previousQueries = queryClient.getQueriesData<ListLearningAreasResponse>({
                queryKey: curriculumKeys.learningAreas.all(),
            });

            queryClient.setQueriesData<ListLearningAreasResponse>(
                { queryKey: curriculumKeys.learningAreas.all() },
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
            queryClient.invalidateQueries({
                queryKey: curriculumKeys.learningAreas.all(),
            });
        },
    });
}

// ─── Hooks: Strands ───────────────────────────────────────────────────────

/** Create a strand with optimistic update. Invalidates the parent learning area tree. */
export function useCreateStrand() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: CreateStrandPayload) => createStrand(data),
        onMutate: async (data) => {
            await queryClient.cancelQueries({
                queryKey: curriculumKeys.learningAreas.tree(data.learning_area_id),
            });
            const previousQueries = queryClient.getQueriesData({
                queryKey: curriculumKeys.learningAreas.tree(data.learning_area_id),
            });
            return { previousQueries };
        },
        onError: (err, _data, context) => {
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }
            toast.error(getErrorMessage(err));
        },
        onSettled: (_data, _err, variables) => {
            queryClient.invalidateQueries({
                queryKey: curriculumKeys.learningAreas.tree(variables.learning_area_id),
            });
        },
    });
}

/** Update a strand with optimistic update. */
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
        onMutate: async ({ id, data, learningAreaId }) => {
            await queryClient.cancelQueries({
                queryKey: curriculumKeys.learningAreas.tree(learningAreaId),
            });
            const previousQueries = queryClient.getQueriesData<LearningAreaTree>({
                queryKey: curriculumKeys.learningAreas.tree(learningAreaId),
            });

            queryClient.setQueriesData<LearningAreaTree>(
                { queryKey: curriculumKeys.learningAreas.tree(learningAreaId) },
                (old) => {
                    if (!old) return old;
                    return {
                        ...old,
                        strands: old.strands.map((s) => (s.id === id ? { ...s, ...data } : s)),
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
        onSettled: (_data, _err, vars) => {
            queryClient.invalidateQueries({
                queryKey: curriculumKeys.learningAreas.tree(vars.learningAreaId),
            });
        },
    });
}

/** Delete a strand with optimistic removal from the learning area tree. */
export function useDeleteStrand() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, learningAreaId }: { id: string; learningAreaId: string }) =>
            deleteStrand(id).then(() => ({ learningAreaId })),
        onMutate: async ({ id, learningAreaId }) => {
            await queryClient.cancelQueries({
                queryKey: curriculumKeys.learningAreas.tree(learningAreaId),
            });
            const previousQueries = queryClient.getQueriesData<LearningAreaTree>({
                queryKey: curriculumKeys.learningAreas.tree(learningAreaId),
            });

            queryClient.setQueriesData<LearningAreaTree>(
                { queryKey: curriculumKeys.learningAreas.tree(learningAreaId) },
                (old) => {
                    if (!old) return old;
                    return {
                        ...old,
                        strands: old.strands.filter((s) => s.id !== id),
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
        onSettled: (_data, _err, vars) => {
            queryClient.invalidateQueries({
                queryKey: curriculumKeys.learningAreas.tree(vars.learningAreaId),
            });
        },
    });
}

// ─── Hooks: Sub-Strands ───────────────────────────────────────────────────

/** Create a sub-strand with optimistic update. Invalidates the parent learning area tree. */
export function useCreateSubStrand() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: CreateSubStrandPayload) => createSubStrand(data),
        onMutate: async () => {
            await queryClient.cancelQueries({
                queryKey: curriculumKeys.learningAreas.all(),
            });
            const previousQueries = queryClient.getQueriesData({
                queryKey: curriculumKeys.learningAreas.all(),
            });
            return { previousQueries };
        },
        onError: (err, _data, context) => {
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }
            toast.error(getErrorMessage(err));
        },
        onSettled: () => {
            queryClient.invalidateQueries({
                queryKey: curriculumKeys.learningAreas.all(),
            });
        },
    });
}

/** Update a sub-strand with optimistic update. */
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
        onMutate: async ({ id, data, learningAreaId }) => {
            await queryClient.cancelQueries({
                queryKey: curriculumKeys.learningAreas.tree(learningAreaId),
            });
            const previousQueries = queryClient.getQueriesData<LearningAreaTree>({
                queryKey: curriculumKeys.learningAreas.tree(learningAreaId),
            });

            queryClient.setQueriesData<LearningAreaTree>(
                { queryKey: curriculumKeys.learningAreas.tree(learningAreaId) },
                (old) => {
                    if (!old) return old;
                    return {
                        ...old,
                        strands: old.strands.map((strand) => ({
                            ...strand,
                            sub_strands: strand.sub_strands.map((ss) =>
                                ss.id === id ? { ...ss, ...data } : ss
                            ),
                        })),
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
        onSettled: (_data, _err, vars) => {
            queryClient.invalidateQueries({
                queryKey: curriculumKeys.learningAreas.tree(vars.learningAreaId),
            });
        },
    });
}

/** Delete a sub-strand with optimistic removal from the learning area tree. */
export function useDeleteSubStrand() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, learningAreaId }: { id: string; learningAreaId: string }) =>
            deleteSubStrand(id).then(() => ({ learningAreaId })),
        onMutate: async ({ id, learningAreaId }) => {
            await queryClient.cancelQueries({
                queryKey: curriculumKeys.learningAreas.tree(learningAreaId),
            });
            const previousQueries = queryClient.getQueriesData<LearningAreaTree>({
                queryKey: curriculumKeys.learningAreas.tree(learningAreaId),
            });

            queryClient.setQueriesData<LearningAreaTree>(
                { queryKey: curriculumKeys.learningAreas.tree(learningAreaId) },
                (old) => {
                    if (!old) return old;
                    return {
                        ...old,
                        strands: old.strands.map((strand) => ({
                            ...strand,
                            sub_strands: strand.sub_strands.filter((ss) => ss.id !== id),
                        })),
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
        onSettled: (_data, _err, vars) => {
            queryClient.invalidateQueries({
                queryKey: curriculumKeys.learningAreas.tree(vars.learningAreaId),
            });
        },
    });
}

// ─── Hooks: Performance Indicators ────────────────────────────────────────

/** Create a performance indicator with optimistic update. Invalidates the parent tree. */
export function useCreatePerformanceIndicator() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: CreatePerformanceIndicatorPayload) => createPerformanceIndicator(data),
        onMutate: async () => {
            await queryClient.cancelQueries({
                queryKey: curriculumKeys.learningAreas.all(),
            });
            const previousQueries = queryClient.getQueriesData({
                queryKey: curriculumKeys.learningAreas.all(),
            });
            return { previousQueries };
        },
        onError: (err, _data, context) => {
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }
            toast.error(getErrorMessage(err));
        },
        onSettled: () => {
            queryClient.invalidateQueries({
                queryKey: curriculumKeys.learningAreas.all(),
            });
        },
    });
}

/** Update a performance indicator with optimistic update. */
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
        onMutate: async ({ id, data, learningAreaId }) => {
            await queryClient.cancelQueries({
                queryKey: curriculumKeys.learningAreas.tree(learningAreaId),
            });
            const previousQueries = queryClient.getQueriesData<LearningAreaTree>({
                queryKey: curriculumKeys.learningAreas.tree(learningAreaId),
            });

            queryClient.setQueriesData<LearningAreaTree>(
                { queryKey: curriculumKeys.learningAreas.tree(learningAreaId) },
                (old) => {
                    if (!old) return old;
                    return {
                        ...old,
                        strands: old.strands.map((strand) => ({
                            ...strand,
                            sub_strands: strand.sub_strands.map((ss) => ({
                                ...ss,
                                performance_indicators: ss.performance_indicators.map((pi) =>
                                    pi.id === id ? { ...pi, ...data } : pi
                                ),
                            })),
                        })),
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
        onSettled: (_data, _err, vars) => {
            queryClient.invalidateQueries({
                queryKey: curriculumKeys.learningAreas.tree(vars.learningAreaId),
            });
        },
    });
}

/** Delete a performance indicator with optimistic removal from the learning area tree. */
export function useDeletePerformanceIndicator() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, learningAreaId }: { id: string; learningAreaId: string }) =>
            deletePerformanceIndicator(id).then(() => ({ learningAreaId })),
        onMutate: async ({ id, learningAreaId }) => {
            await queryClient.cancelQueries({
                queryKey: curriculumKeys.learningAreas.tree(learningAreaId),
            });
            const previousQueries = queryClient.getQueriesData<LearningAreaTree>({
                queryKey: curriculumKeys.learningAreas.tree(learningAreaId),
            });

            queryClient.setQueriesData<LearningAreaTree>(
                { queryKey: curriculumKeys.learningAreas.tree(learningAreaId) },
                (old) => {
                    if (!old) return old;
                    return {
                        ...old,
                        strands: old.strands.map((strand) => ({
                            ...strand,
                            sub_strands: strand.sub_strands.map((ss) => ({
                                ...ss,
                                performance_indicators: ss.performance_indicators.filter(
                                    (pi) => pi.id !== id
                                ),
                            })),
                        })),
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
        onSettled: (_data, _err, vars) => {
            queryClient.invalidateQueries({
                queryKey: curriculumKeys.learningAreas.tree(vars.learningAreaId),
            });
        },
    });
}
