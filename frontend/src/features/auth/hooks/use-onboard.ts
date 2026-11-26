import { useMutation } from "@tanstack/react-query";
import { onboard } from "../api/auth.api";
import { authKeys } from "../api/auth.keys";
import type { OnboardInput, OnboardResponse } from "../types";

/**
 * useOnboard wraps onboard in a TanStack Query mutation.
 * Returns the mutation trigger, status, and error.
 */
export function useOnboard() {
  return useMutation<OnboardResponse, Error, OnboardInput>({
    mutationKey: authKeys.onboard(),
    mutationFn: onboard,
  });
}
