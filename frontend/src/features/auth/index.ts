export { useRequestOTP } from "./hooks/use-request-otp";
export { useVerifyOTP } from "./hooks/use-verify-otp";
export { useOnboard } from "./hooks/use-onboard";
export { requestOTP, verifyOTP, onboard } from "./api/auth.api";
export { authKeys } from "./api/auth.keys";
export type {
  RequestOTPInput,
  RequestOTPResponse,
  VerifyOTPInput,
  VerifyOTPResponse,
  OnboardInput,
  OnboardResponse,
  AuthError,
} from "./types";
export type {
  RequestOTPFormValues,
  VerifyOTPFormValues,
} from "./schemas/otp.schema";
export type { OnboardFormValues } from "./schemas/onboard.schema";
