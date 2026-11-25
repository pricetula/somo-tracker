import type { components } from "@/lib/generated/api";

/**
 * Generated API contract types for the auth module.
 * These mirror the backend wire format (snake_case) and are regenerated
 * from backend/docs/swagger.json via `make generate-frontend-types`.
 * Do not edit by hand — changes will be overwritten.
 */
export type RequestOTPRequestBody = components["schemas"]["handler.RequestOTPRequest"];
export type RequestOTPResponseBody = components["schemas"]["handler.RequestOTPResponse"];
export type VerifyOTPRequestBody = components["schemas"]["handler.VerifyOTPRequest"];
export type VerifyOTPResponseBody = components["schemas"]["handler.VerifyOTPResponse"];
export type OnboardRequestBody = components["schemas"]["handler.OnboardRequest"];
export type SessionResponseBody = components["schemas"]["handler.SessionResponse"];
export type ChallengeRequestBody = components["schemas"]["handler.ChallengeRequest"];
export type ChallengeResponseBody = components["schemas"]["handler.ChallengeResponse"];
