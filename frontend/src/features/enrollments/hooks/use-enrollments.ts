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

export function useDeleteEnrollment(classId: string) {
    const queryClient = useQueryClient();
    return useMutation({
        mutationKey: [...enrollmentKeys.list(classId), "delete"],
        mutationFn: (id: string) => deleteEnrollment(id),
        onMutate: async (id) => {
            await queryClient.cancelQueries({ queryKey: enrollmentKeys.list(classId) });
            const previousData = queryClient.getQueriesData({
                queryKey: enrollmentKeys.list(classId),
            });
            queryClient.setQueriesData(
                { queryKey: enrollmentKeys.list(classId) },
                (old: unknown) => {
                    if (!old || typeof old !== "object") return old;
                    const data = old as
                        | { pages?: { items: { id: string }[]; total?: number }[] }
                        | { items: { id: string }[]; total?: number };
                    if ("pages" in data && Array.isArray(data.pages)) {
                        return {
                            ...data,
                            pages: data.pages.map((page) => ({
                                ...page,
                                items:
                                    page.items?.filter((item: { id: string }) => item.id !== id) ??
                                    [],
                                total: typeof page.total === "number" ? page.total - 1 : page.total,
                            })),
                        };
                    }
                    if ("items" in data && Array.isArray(data.items)) {
                        return {
                            ...data,
                            items: data.items.filter((item: { id: string }) => item.id !== id),
                            total: typeof data.total === "number" ? data.total - 1 : data.total,
                        };
                    }
                    return old;
                }
            );
            return { previousData };
        },
        onError: (err, _id, context) => {
            if (context?.previousData) {
                context.previousData.forEach(([key, data]) => {
                    queryClient.setQueryData(key, data);
                });
            }
            toast.error(getErrorMessage(err));
        },
        onSuccess: () => {
            toast.success("Enrollment deleted");
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: enrollmentKeys.list(classId) });
        },
    });
}
