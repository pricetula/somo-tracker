export interface RequestOTPInput {
  email: string;
  turnstileToken: string | null;
}

export interface RequestOTPResponse {
  ok: boolean;
  message: string;
  challengeId: string;
}

export interface VerifyOTPInput {
  challengeId: string;
  code: string;
}

export interface VerifyOTPResponse {
  ok: boolean;
  message: string;
  needsOnboarding: boolean;
}

export interface AuthError {
  code: string;
  message: string;
}
