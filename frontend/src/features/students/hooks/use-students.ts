"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { deleteStudents } from "@/lib/api/students";
import { getErrorMessage } from "@/lib/errors";
import { toast } from "sonner";

export const studentKeys = {
    list: ["students"] as const,
};

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
            queryClient.invalidateQueries({ queryKey: studentKeys.list });
        },
        onSuccess: () => {
            toast.success("Students deleted");
        },
    });
}
