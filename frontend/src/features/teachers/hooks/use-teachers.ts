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
        onSuccess: () => {
            toast.success("Teachers deleted");
            queryClient.invalidateQueries({ queryKey: teacherKeys.list });
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}
