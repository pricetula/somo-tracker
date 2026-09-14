/**
 * Guardians feature — hooks for listing and deleting guardians.
 */

"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { deleteGuardians } from "@/lib/api/guardians";
import { getErrorMessage } from "@/lib/errors";
import { toast } from "sonner";

export const guardianKeys = {
    list: ["guardians"] as const,
};

export function useDeleteGuardians() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationKey: [...guardianKeys.list, "delete"],
        mutationFn: (userIds: string[]) => deleteGuardians(userIds),
        onSuccess: () => {
            toast.success("Guardians deleted");
            queryClient.invalidateQueries({ queryKey: guardianKeys.list });
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}
