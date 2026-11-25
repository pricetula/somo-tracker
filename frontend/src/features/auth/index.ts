export { useRequestOTP } from "./hooks/use-request-otp";
export { useVerifyOTP } from "./hooks/use-verify-otp";
export { requestOTP, verifyOTP } from "./api/auth.api";
export { authKeys } from "./api/auth.keys";
export type {
  RequestOTPInput,
  RequestOTPResponse,
  VerifyOTPInput,
  VerifyOTPResponse,
  AuthError,
} from "./types";
export type {
  RequestOTPFormValues,
  VerifyOTPFormValues,
} from "./schemas/otp.schema";
