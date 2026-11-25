import { useMutation } from "@tanstack/react-query";
import { verifyOTP } from "../api/auth.api";
import { authKeys } from "../api/auth.keys";
import type { VerifyOTPInput, VerifyOTPResponse } from "../types";

/**
 * useVerifyOTP wraps verifyOTP in a TanStack Query mutation.
 * Returns the mutation trigger, status, and error.
 */
export function useVerifyOTP() {
  return useMutation<VerifyOTPResponse, Error, VerifyOTPInput>({
    mutationKey: authKeys.otpVerify(),
    mutationFn: verifyOTP,
  });
}
