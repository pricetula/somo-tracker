"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { getErrorMessage } from "@/lib/errors";
import { toast } from "sonner";
import {
    listByClass,
    listUnassigned,
    createBatch,
    updateEnrollment,
    deleteEnrollment,
} from "../services/api";
import type { CreateEnrollmentRequest } from "../types/enrollment";

export const enrollmentKeys = {
    list: (classId: string) => ["enrollments", classId] as const,
    unassigned: ["unassigned-students"] as const,
};

export function useEnrollmentsByClass(classId: string) {
    return useQuery({
        queryKey: enrollmentKeys.list(classId),
        queryFn: () => listByClass(classId),
        enabled: !!classId,
        staleTime: 5 * 60 * 1000,
    });
}

export function useUnassignedStudents() {
    return useQuery({
        queryKey: enrollmentKeys.unassigned,
        queryFn: () => listUnassigned(),
        staleTime: 5 * 60 * 1000,
    });
}

export function useCreateEnrollments(classId: string) {
    const queryClient = useQueryClient();
    return useMutation({
        mutationKey: [...enrollmentKeys.list(classId), "create"],
        mutationFn: (data: CreateEnrollmentRequest[]) => createBatch(classId, data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: enrollmentKeys.list(classId) });
            queryClient.invalidateQueries({ queryKey: enrollmentKeys.unassigned });
            toast.success("Students enrolled");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

export function useUpdateEnrollment() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationKey: ["enrollments", "update"],
        mutationFn: ({ id, data }: { id: string; data: Partial<CreateEnrollmentRequest> }) =>
            updateEnrollment(id, data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["enrollments"] });
            toast.success("Enrollment updated");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

export function useDeleteEnrollment() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationKey: ["enrollments", "delete"],
        mutationFn: (id: string) => deleteEnrollment(id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["enrollments"] });
            toast.success("Enrollment deleted");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}
