import { useMutation } from "@tanstack/react-query";
import { requestOTP } from "../api/auth.api";
import { authKeys } from "../api/auth.keys";
import type { RequestOTPInput, RequestOTPResponse } from "../types";

/**
 * useRequestOTP wraps requestOTP in a TanStack Query mutation.
 * Returns the mutation trigger, status, and error.
 */
export function useRequestOTP() {
  return useMutation<RequestOTPResponse, Error, RequestOTPInput>({
    mutationKey: authKeys.otpRequest(),
    mutationFn: requestOTP,
  });
}
