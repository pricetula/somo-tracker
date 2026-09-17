/**
 * Finance feature — hooks for listing and deleting finance.
 */

"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { deleteFinance } from "@/lib/api/finance";
import { getErrorMessage } from "@/lib/errors";
import { toast } from "sonner";

export const financeKeys = {
    list: ["finance"] as const,
};

export function useDeleteFinance() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationKey: [...financeKeys.list, "delete"],
        mutationFn: (userIds: string[]) => deleteFinance(userIds),
        async onMutate(userIds) {
            await queryClient.cancelQueries({ queryKey: financeKeys.list });
            const previous = queryClient.getQueriesData({
                queryKey: financeKeys.list,
                exact: false,
            });
            queryClient.setQueriesData(
                { queryKey: financeKeys.list, exact: false },
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
            queryClient.invalidateQueries({ queryKey: financeKeys.list });
        },
        onSuccess: () => {
            toast.success("Finance deleted");
        },
    });
}
