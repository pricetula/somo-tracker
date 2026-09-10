/**
 * TanStack Query hooks — School Registration (new implementation).
 * Previous hooks for list/update/delete/seed are deprecated.
 */

"use client";

import { useMutation } from "@tanstack/react-query";
import { toast } from "sonner";
import { registerSchool } from "@/lib/api/schools";
import type { RegisterSchoolPayload, RegisterSchoolResponse } from "@/lib/api/schools";
import { getErrorMessage } from "@/lib/errors";

export const schoolRegistrationKeys = {
    register: ["school-registration"] as const,
};

/** Mutation: register a new school with atomic user + school + membership creation. */
export function useRegisterSchool() {
    return useMutation<RegisterSchoolResponse, Error, RegisterSchoolPayload>({
        mutationKey: schoolRegistrationKeys.register,
        mutationFn: registerSchool,
        onSuccess: (data) => {
            toast.success(data.message ?? "School registered successfully");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}
