/**
 * TanStack Query hooks — Create School (admin-only).
 */

"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { createSchool } from "@/lib/api/schools";
import type { CreateSchoolPayload, CreateSchoolResponse } from "@/lib/api/schools";
import { getErrorMessage } from "@/lib/errors";
import { schoolsQueryKeys } from "./use-schools-list";

export const createSchoolKeys = {
    create: ["school-create"] as const,
};

/** Mutation: create a new school with admin verification and setup. */
export function useCreateSchool() {
    const queryClient = useQueryClient();
    return useMutation<CreateSchoolResponse, Error, CreateSchoolPayload>({
        mutationKey: createSchoolKeys.create,
        mutationFn: createSchool,
        onSuccess: (data) => {
            toast.success(data.message ?? "School created successfully");
            queryClient.invalidateQueries({ queryKey: schoolsQueryKeys.list });
            queryClient.invalidateQueries({ queryKey: ["me"] });
        },
        onError: (err) => {
            const msg = getErrorMessage(err);
            if (msg.toLowerCase().includes("duplicate key")) {
                toast.error(
                    "You already have an active school membership. Deactivate your current school first."
                );
            } else {
                toast.error(msg);
            }
        },
    });
}
