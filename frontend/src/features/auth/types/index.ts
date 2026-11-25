import type {
  OnboardRequestBody,
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

export interface OnboardInput {
  organizationName: string;
  schoolName: string;
  userName: string;
}

/**
 * API response types (camelCase) used by the frontend.
 * The backend wire format (snake_case) is defined in the generated types
 * and mapped in auth.api.ts. These are regenerated from
 * backend/docs/swagger.json via `make generate-frontend-types`.
 *
 * Note: OnboardResponseBody is defined locally because the backend returns
 * an anonymous fiber.Map (no named schema in the OpenAPI spec).
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

export interface OnboardResponse {
  ok: boolean;
  message: string;
  needsOnboarding: boolean;
}

/**
 * Generated wire-format types (snake_case) from the OpenAPI contract.
 * Re-exported for reference and for use in fetchers that need the raw shape.
 */
export type {
  OnboardRequestBody,
  RequestOTPResponseBody,
  VerifyOTPResponseBody,
};

export interface AuthError {
  code: string;
  message: string;
}

/**
 * Onboard response wire format (snake_case). Defined locally because the
 * backend returns an anonymous object with no named OpenAPI schema.
 */
export interface OnboardResponseBody {
  ok?: boolean;
  message?: string;
  needs_onboarding?: boolean;
}
