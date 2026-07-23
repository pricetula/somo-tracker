/**
 * React Query hooks for the Class Teachers feature.
 */
"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
    createClassTeacher,
    deleteClassTeacher,
    getClassTeacher,
    listClassTeachersByClass,
    listClassTeachersByTeacher,
} from "@/lib/api/classteachers";
import type { CreateClassTeacherPayload } from "@/lib/api/classteachers";

// ─── Query Keys ───────────────────────────────────────────────────────────

export const classTeacherKeys = {
    all: ["class-teachers"] as const,
    byClass: (classId: string) => ["class-teachers", "by-class", classId] as const,
    byTeacher: (userId: string) => ["class-teachers", "by-teacher", userId] as const,
    detail: (id: string) => ["class-teachers", id] as const,
};

// ─── Queries ──────────────────────────────────────────────────────────────

/** List all teacher assignments for a class. */
export function useClassTeachersByClass(classId: string) {
    return useQuery({
        queryKey: classTeacherKeys.byClass(classId),
        queryFn: () => listClassTeachersByClass(classId),
        enabled: !!classId,
    });
}

/** List all class assignments for a teacher. */
export function useClassTeachersByTeacher(userId: string) {
    return useQuery({
        queryKey: classTeacherKeys.byTeacher(userId),
        queryFn: () => listClassTeachersByTeacher(userId),
        enabled: !!userId,
    });
}

/** Get a single assignment. */
export function useClassTeacher(id: string) {
    return useQuery({
        queryKey: classTeacherKeys.detail(id),
        queryFn: () => getClassTeacher(id),
        enabled: !!id,
    });
}

// ─── Mutations ────────────────────────────────────────────────────────────

/** Assign a teacher to a class. */
export function useCreateClassTeacher() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: (data: CreateClassTeacherPayload) => createClassTeacher(data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: classTeacherKeys.all });
        },
    });
}

/** Remove a class teacher assignment. */
export function useDeleteClassTeacher() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => deleteClassTeacher(id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: classTeacherKeys.all });
        },
    });
}
