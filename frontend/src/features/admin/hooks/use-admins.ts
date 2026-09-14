/**
 * Admins feature — hooks for listing and deleting admins.
 */

"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { deleteAdmins } from "@/lib/api/admins";
import { getErrorMessage } from "@/lib/errors";
import { toast } from "sonner";

export const adminKeys = {
    list: ["admins"] as const,
};

export function useDeleteAdmins() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationKey: [...adminKeys.list, "delete"],
        mutationFn: (userIds: string[]) => deleteAdmins(userIds),
        onSuccess: () => {
            toast.success("Admins deleted");
            queryClient.invalidateQueries({ queryKey: adminKeys.list });
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}
