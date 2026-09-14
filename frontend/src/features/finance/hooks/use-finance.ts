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
        onSuccess: () => {
            toast.success("Finance deleted");
            queryClient.invalidateQueries({ queryKey: financeKeys.list });
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}
