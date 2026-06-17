/**
 * TanStack Query hooks for students.
 */

"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import {
    listStudents,
    createStudent,
    importStudentCSV,
    type CreateStudentPayload,
    type ListStudentsResponse,
} from "@/lib/api/students";
import { getApiErrorMessage } from "@/lib/api/auth";

// ─── Query keys ───────────────────────────────────────────────────────────

export const studentKeys = {
    all: ["students"] as const,
    list: (filters: { page?: number; per_page?: number; search?: string }) =>
        ["students", "list", filters] as const,
};

// ─── Options type ──────────────────────────────────────────────────────────

interface UseStudentsOptions {
    page?: number;
    per_page?: number;
    search?: string;
    enabled?: boolean;
}

// ─── Hooks ─────────────────────────────────────────────────────────────────

/** Fetch students with pagination and search. */
export function useStudents(opts: UseStudentsOptions = {}) {
    const { page = 1, per_page = 50, search = "", enabled = true } = opts;

    return useQuery<ListStudentsResponse>({
        queryKey: studentKeys.list({ page, per_page, search }),
        queryFn: () => listStudents({ page, per_page, search }),
        placeholderData: (prev) => prev,
        enabled,
    });
}

/** Create a single student manually. Invalidates list cache on success. */
export function useCreateStudent() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: CreateStudentPayload) => createStudent(payload),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: studentKeys.all });
            toast.success("Student added", {
                description: "The student has been created successfully.",
            });
        },
        onError: (err) => {
            toast.error("Failed to add student", {
                description: getApiErrorMessage(err),
            });
        },
    });
}

/** Import students via CSV upload. Returns the import_id for SSE tracking. */
export function useImportCSV() {
    return useMutation({
        mutationFn: (file: File) => importStudentCSV(file),
        onError: (err) => {
            toast.error("CSV upload failed", {
                description: getApiErrorMessage(err),
            });
        },
    });
}
