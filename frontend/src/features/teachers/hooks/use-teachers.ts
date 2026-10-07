/**
 * Teachers feature — hooks for listing and deleting teachers.
 */

"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { deleteTeachers } from "@/lib/api/teachers";
import { getErrorMessage } from "@/lib/errors";
import { toast } from "sonner";

export const teacherKeys = {
    list: ["teachers"] as const,
};

export function useDeleteTeachers() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationKey: [...teacherKeys.list, "delete"],
        mutationFn: (userIds: string[]) => deleteTeachers(userIds),
        async onMutate(userIds) {
            await queryClient.cancelQueries({ queryKey: teacherKeys.list });
            const previous = queryClient.getQueriesData({
                queryKey: teacherKeys.list,
                exact: false,
            });
            queryClient.setQueriesData(
                { queryKey: teacherKeys.list, exact: false },
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
                                            return !userIds.includes(String(it.user_id ?? ""));
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
                            return !userIds.includes(String(it.user_id ?? ""));
                        });
                        return {
                            ...data,
                            items: filtered,
                            total: Math.max(
                                0,
                                (typeof data.total === "number" ? data.total : 0) - userIds.length
                            ),
                        };
                    }
                    if (Array.isArray(oldData)) {
                        return (oldData as unknown[]).filter((i) => {
                            const it = i as Record<string, unknown>;
                            return !userIds.includes(String(it.user_id ?? ""));
                        });
                    }
                    return oldData;
                }
            );
            return { previous };
        },
        onError(err, _userIds, context) {
            if (context?.previous) {
                context.previous.forEach(([key, data]) => {
                    queryClient.setQueryData(key, data);
                });
            }
            toast.error(getErrorMessage(err));
        },
        onSettled() {
            queryClient.invalidateQueries({ queryKey: teacherKeys.list });
        },
        onSuccess: () => {
            toast.success("Teachers deleted");
        },
    });
}
