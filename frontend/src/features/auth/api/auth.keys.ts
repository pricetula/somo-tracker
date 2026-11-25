export const authKeys = {
  all: ["auth"] as const,
  otpRequest: () => [...authKeys.all, "otp-request"] as const,
  otpVerify: () => [...authKeys.all, "otp-verify"] as const,
  onboard: () => [...authKeys.all, "onboard"] as const,
};
