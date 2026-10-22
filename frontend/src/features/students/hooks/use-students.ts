"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
    deleteStudents,
    listStudents,
    bulkAssignStudentsToClass,
    getStudent,
    updateStudent,
    type UpdateStudentRequest,
} from "@/lib/api/students";
import { getErrorMessage } from "@/lib/errors";
import { toast } from "sonner";

function mapStudentFiltersToParams(filters?: Record<string, string | string[]>) {
    const classId = typeof filters?.class === "string" ? filters.class : undefined;
    const withoutClass = filters?.without_class === "true";
    const withoutGuardian = filters?.without_guardian === "true";
    return { class_id: classId, without_class: withoutClass, without_guardian: withoutGuardian };
}

export const studentKeys = {
    list: ["students"] as const,
    detail: (id: string) => ["students", id] as const,
};

export function useStudent(id: string) {
    return useQuery({
        queryKey: studentKeys.detail(id),
        queryFn: () => getStudent(id),
        enabled: !!id,
    });
}

export function useUpdateStudent(id: string) {
    const queryClient = useQueryClient();
    return useMutation({
        mutationKey: [...studentKeys.detail(id), "update"],
        mutationFn: (data: UpdateStudentRequest) => updateStudent(id, data),
        onSuccess: () => {
            toast.success("Student updated");
            queryClient.invalidateQueries({ queryKey: studentKeys.detail(id) });
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

export function studentsListQueryFn(params: {
    page?: number;
    limit?: number;
    search?: string;
    filters?: Record<string, string | string[]>;
}) {
    // map filters via component helper import to avoid circular deps
    // For now, forward params directly; filter mapping is handled by caller
    const { class_id, without_class, without_guardian } = params.filters
        ? mapStudentFiltersToParams(params.filters)
        : { class_id: undefined, without_class: undefined, without_guardian: undefined };
    return listStudents({
        page: params.page,
        limit: params.limit,
        search: params.search,
        class_id,
        without_class,
        without_guardian,
    });
}

export function useAssignStudents() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationKey: [...studentKeys.list, "assign"],
        mutationFn: ({ classId, studentIds }: { classId: string; studentIds: string[] }) =>
            bulkAssignStudentsToClass(classId, studentIds),
        onSuccess: () => {
            toast.success("Students assigned to class");
            queryClient.invalidateQueries({ queryKey: studentKeys.list });
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

export function useDeleteStudents() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationKey: [...studentKeys.list, "delete"],
        mutationFn: (studentIds: string[]) => deleteStudents(studentIds),
        async onMutate(studentIds) {
            await queryClient.cancelQueries({ queryKey: studentKeys.list });
            const previous = queryClient.getQueriesData({
                queryKey: studentKeys.list,
                exact: false,
            });
            queryClient.setQueriesData(
                { queryKey: studentKeys.list, exact: false },
                (oldData: unknown) => {
                    if (!oldData) return oldData;
                    const data = oldData as Record<string, unknown>;
                    if (Array.isArray(data.pages)) {
                        return {
                            ...data,
                            pages: (data.pages as unknown[]).map((page) => {
                                const p = page as Record<string, unknown>;
                                const items = (p.items ?? p) as unknown[];
                                if (Array.isArray(items)) {
                                    return {
                                        ...p,
                                        items: items.filter((i) => {
                                            const it = i as Record<string, unknown>;
                                            return !studentIds.includes(
                                                String(it.student_id ?? "")
                                            );
                                        }),
                                    };
                                }
                                return p;
                            }),
                        };
                    }
                    if ("items" in data && Array.isArray(data.items)) {
                        const items = data.items as unknown[];
                        const filtered = items.filter((i) => {
                            const it = i as Record<string, unknown>;
                            return !studentIds.includes(String(it.student_id ?? ""));
                        });
                        return {
                            ...data,
                            items: filtered,
                            total: Math.max(
                                0,
                                (typeof data.total === "number" ? data.total : 0) -
                                    studentIds.length
                            ),
                        };
                    }
                    if (Array.isArray(oldData)) {
                        return (oldData as unknown[]).filter((i) => {
                            const it = i as Record<string, unknown>;
                            return !studentIds.includes(String(it.student_id ?? ""));
                        });
                    }
                    return oldData;
                }
            );
            return { previous };
        },
        onError(err, _studentIds, context) {
            if (context?.previous) {
                context.previous.forEach(([key, data]) => {
                    queryClient.setQueryData(key, data);
                });
            }
            toast.error(getErrorMessage(err));
        },
        onSettled() {
            // queryClient.invalidateQueries({ queryKey: studentKeys.list });
        },
        onSuccess: () => {
            toast.success("Students deleted");
        },
    });
}
