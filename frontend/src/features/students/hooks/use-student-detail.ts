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
    type CreateStudentsResponse,
    type UpdateStudentPayload,
    type CreateEnrollmentPayload,
    type BatchEnrollRequest,
    type BatchEnrollResponse,
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

/** Create a single student (convenience wrapper around batch). */
export function useCreateStudent() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: CreateStudentPayload) => createStudent(data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: studentKeys.all });
            toast.success("Student created");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

// ─── Mutations: Batch Create ──────────────────────────────────────────────

/** Create multiple students in one request (batch). */
export function useCreateStudents() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: CreateStudentsPayload) => createStudents(data),
        onSuccess: (result: CreateStudentsResponse) => {
            queryClient.invalidateQueries({ queryKey: studentKeys.all });
            toast.success(
                `${result.ids.length} student${result.ids.length === 1 ? "" : "s"} created`
            );
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

// ─── Mutations: Update ────────────────────────────────────────────────────

/** Update a student's demographics. */
export function useUpdateStudent() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, data }: { id: string; data: UpdateStudentPayload }) =>
            updateStudent(id, data),
        onSuccess: (_, variables) => {
            queryClient.invalidateQueries({ queryKey: studentKeys.all });
            queryClient.invalidateQueries({
                queryKey: studentDetailKeys.detail(variables.id),
            });
            toast.success("Student updated");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

// ─── Mutations: Enrollments ───────────────────────────────────────────────

/** Enroll a student in a class for a term. */
export function useCreateEnrollment() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ studentId, data }: { studentId: string; data: CreateEnrollmentPayload }) =>
            createEnrollment(studentId, data),
        onSuccess: (_, variables) => {
            queryClient.invalidateQueries({
                queryKey: studentDetailKeys.detail(variables.studentId),
            });
            toast.success("Student enrolled");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

/** Batch enroll multiple students in a class for the current academic term. */
export function useBatchEnrollStudents() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: BatchEnrollRequest) => batchEnrollStudents(data),
        onSuccess: (result: BatchEnrollResponse) => {
            queryClient.invalidateQueries({ queryKey: ["students"] });
            toast.success(
                `${result.ids.length} student${result.ids.length === 1 ? "" : "s"} enrolled`
            );
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}
