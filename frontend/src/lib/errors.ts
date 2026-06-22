/**
 * Error handling utilities for the frontend.
 *
 * All catch blocks must use getErrorMessage(err) instead of (err as Error).message.
 *
 * Location documented in:
 *   - frontend/AGENTS.md (Error Handling section)
 *   - root AGENTS.md (Error Handling section)
 */

import { ApiError } from "./api/client";

/**
 * Safely extracts a human-readable error message from any thrown value.
 * - ApiError instances: returns err.message
 * - plain Error instances: returns err.message
 * - unknown throws (strings, objects, null): returns a safe fallback
 *
 * Never throws. Never returns undefined.
 */
export function getErrorMessage(err: unknown): string {
    if (err instanceof ApiError) {
        return err.message;
    }
    if (err instanceof Error) {
        return err.message;
    }
    if (typeof err === "string") {
        return err;
    }
    return "An unexpected error occurred";
}

/**
 * Type guard for ApiError.
 */
export function isApiError(err: unknown): err is ApiError {
    return err instanceof ApiError;
}
