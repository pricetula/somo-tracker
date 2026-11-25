import type {
  RequestOTPResponseBody,
  VerifyOTPResponseBody,
} from "./generated";

/**
 * Frontend-facing input types (camelCase) used by forms and hooks.
 * These are converted to snake_case wire format in auth.api.ts.
 */
export interface RequestOTPInput {
  email: string;
  turnstileToken: string | null;
}

export interface VerifyOTPInput {
  challengeId: string;
  code: string;
}

/**
 * API response types (camelCase) used by the frontend.
 * The backend wire format (snake_case) is defined in the generated types
 * and mapped in auth.api.ts. These are regenerated from
 * backend/docs/swagger.json via `make generate-frontend-types`.
 */
export interface RequestOTPResponse {
  ok: boolean;
  message: string;
  challengeId: string;
}

export interface VerifyOTPResponse {
  ok: boolean;
  message: string;
  needsOnboarding: boolean;
}

/**
 * Generated wire-format types (snake_case) from the OpenAPI contract.
 * Re-exported for reference and for use in fetchers that need the raw shape.
 */
export type { RequestOTPResponseBody, VerifyOTPResponseBody };

export interface AuthError {
  code: string;
  message: string;
}
