/**
 * TanStack Query mutation for seeding default CBC curriculum.
 */

"use client";

import { useMutation } from "@tanstack/react-query";
import { toast } from "sonner";

import { seedDefaultCBC } from "@/lib/api/curriculum";
import { getErrorMessage } from "@/lib/errors";

export function useSeedDefaultCBC() {
    return useMutation({
        mutationFn: seedDefaultCBC,
        onSuccess: () => {
            toast.success("Default CBC curriculum seeded successfully.");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}
