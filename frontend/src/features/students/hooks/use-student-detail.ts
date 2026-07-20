/**
 * TanStack Query hooks for student detail, CRUD, and enrollment operations.
 */

"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import {
    getStudentDetail,
    createStudent,
    createStudents,
    updateStudent,
    createEnrollment,
    batchEnrollStudents,
    type StudentDetailResponse,
    type CreateStudentPayload,
    type CreateStudentsPayload,
    type UpdateStudentPayload,
    type CreateEnrollmentPayload,
    type BatchEnrollRequest,
} from "@/lib/api/students";
import { studentKeys } from "./use-students";
import { getErrorMessage } from "@/lib/errors";

// ─── Query keys ───────────────────────────────────────────────────────────

export const studentDetailKeys = {
    detail: (id: string) => [...studentKeys.all, "detail", id] as const,
    enrollments: (id: string) => [...studentKeys.all, "detail", id, "enrollments"] as const,
};

// ─── Hooks: Detail ────────────────────────────────────────────────────────

/** Fetch student detail with enrollment history. */
export function useStudentDetail(id: string, opts: { enabled?: boolean } = {}) {
    const { enabled = true } = opts;

    return useQuery<StudentDetailResponse>({
        queryKey: studentDetailKeys.detail(id),
        queryFn: () => getStudentDetail(id),
        enabled: enabled && !!id,
    });
}

// ─── Mutations: Create (single) ───────────────────────────────────────────

/** Create a single student with optimistic update. */
export function useCreateStudent() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: CreateStudentPayload) => createStudent(data),
        onMutate: async () => {
            await queryClient.cancelQueries({ queryKey: studentKeys.all });
            const previousQueries = queryClient.getQueriesData({
                queryKey: studentKeys.all,
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
            queryClient.invalidateQueries({ queryKey: studentKeys.all });
        },
    });
}

// ─── Mutations: Batch Create ──────────────────────────────────────────────

/** Create multiple students in one request (batch) with optimistic update. */
export function useCreateStudents() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: CreateStudentsPayload) => createStudents(data),
        onMutate: async () => {
            await queryClient.cancelQueries({ queryKey: studentKeys.all });
            const previousQueries = queryClient.getQueriesData({
                queryKey: studentKeys.all,
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
            queryClient.invalidateQueries({ queryKey: studentKeys.all });
        },
    });
}

// ─── Mutations: Update ────────────────────────────────────────────────────

/** Update a student's demographics with optimistic update. */
export function useUpdateStudent() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, data }: { id: string; data: UpdateStudentPayload }) =>
            updateStudent(id, data),
        onMutate: async ({ id, data }) => {
            await queryClient.cancelQueries({ queryKey: studentKeys.all });
            const previousQueries = queryClient.getQueriesData({
                queryKey: studentKeys.all,
            });

            queryClient.setQueriesData(
                { queryKey: studentKeys.all },
                (old: { items: { id: string }[] } | undefined) => {
                    if (!old) return old;
                    return {
                        ...old,
                        items: old.items.map((item) =>
                            item.id === id ? { ...item, ...data } : item
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
        onSettled: (_data, _err, variables) => {
            queryClient.invalidateQueries({ queryKey: studentKeys.all });
            queryClient.invalidateQueries({
                queryKey: studentDetailKeys.detail(variables.id),
            });
        },
    });
}

// ─── Mutations: Enrollments ───────────────────────────────────────────────

/** Enroll a student in a class for a term with optimistic update. */
export function useCreateEnrollment() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ studentId, data }: { studentId: string; data: CreateEnrollmentPayload }) =>
            createEnrollment(studentId, data),
        onMutate: async ({ studentId }) => {
            await queryClient.cancelQueries({
                queryKey: studentDetailKeys.detail(studentId),
            });
            const previousQueries = queryClient.getQueriesData({
                queryKey: studentDetailKeys.detail(studentId),
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
                queryKey: studentDetailKeys.detail(variables.studentId),
            });
        },
    });
}

/** Batch enroll multiple students with optimistic update. */
export function useBatchEnrollStudents() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: BatchEnrollRequest) => batchEnrollStudents(data),
        onMutate: async () => {
            await queryClient.cancelQueries({ queryKey: studentKeys.all });
            const previousQueries = queryClient.getQueriesData({
                queryKey: studentKeys.all,
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
            queryClient.invalidateQueries({ queryKey: studentKeys.all });
        },
    });
}
