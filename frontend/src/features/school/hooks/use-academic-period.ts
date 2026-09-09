/**
 * TanStack Query mutation — Academic Period creation.
 */

"use client";

import { useMutation } from "@tanstack/react-query";
import { toast } from "sonner";
import { createAcademicPeriod } from "@/lib/api/academic-terms";
import type { AcademicPeriodRequest, CreateAcademicPeriodResponse } from "@/lib/api/academic-terms";
import { getErrorMessage } from "@/lib/errors";

export const academicPeriodMutationKeys = {
    create: ["academic-period", "create"] as const,
};

export function useCreateAcademicPeriod() {
    return useMutation<CreateAcademicPeriodResponse, Error, AcademicPeriodRequest>({
        mutationKey: academicPeriodMutationKeys.create,
        mutationFn: createAcademicPeriod,
        onSuccess: (data) => {
            toast.success(data.message ?? "Academic period created successfully");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}
