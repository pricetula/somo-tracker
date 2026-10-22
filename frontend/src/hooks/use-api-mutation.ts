"use client";

import { useMutation, UseMutationOptions, MutationFunctionContext } from "@tanstack/react-query";
import { ApiError } from "@/lib/api/client";
import { usePeriodError } from "@/components/providers/period-error-provider";

export function useApiMutation<TData, TError = ApiError, TVariables = unknown, TContext = unknown>(
    mutationFn: (variables: TVariables) => Promise<TData>,
    options?: UseMutationOptions<TData, TError, TVariables, TContext> & { redirectTo?: string }
) {
    const { showPeriodError } = usePeriodError();

    return useMutation<TData, TError, TVariables, TContext>({
        mutationFn,
        onError: (error, variables, context) => {
            const err = error as ApiError;
            if (err?.code === "no_next_academic_term") {
                // Trigger global period dialog with optional retry
                // No upcoming period available, hide "Use upcoming period" button
                showPeriodError(
                    async () => {
                        await mutationFn(variables as TVariables);
                    },
                    { showUseUpcoming: false }
                );
                return;
            }
            if (err?.code === "no_active_term_future_exists") {
                // Trigger global period dialog with retry that bypasses future term modal
                const modifiedVars =
                    typeof variables === "object" && variables !== null
                        ? { ...(variables as object), bypassFutureTermModal: true }
                        : variables;
                showPeriodError(
                    async () => {
                        await mutationFn(modifiedVars as TVariables);
                    },
                    { showUseUpcoming: true, redirectTo: options?.redirectTo }
                );
                return;
            }
            options?.onError?.(
                error,
                variables,
                undefined,
                context as unknown as MutationFunctionContext
            );
        },
        ...options,
    });
}
