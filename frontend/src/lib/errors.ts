import { ApiError, isApiError, type FieldError } from './api/errors';

export function getErrorMessage(err: unknown): string {
  if (isApiError(err)) {
    // For 5xx and network errors, surface generic message with request id in UI
    // Here we return the server message for 4xx, generic otherwise
    if (err.status >= 500 || err.status === 0) {
      return 'Something went wrong. Please try again.';
    }
    return err.message;
  }
  if (err instanceof Error) {
    return err.message;
  }
  return 'An unknown error occurred';
}

export function getApiError(err: unknown): ApiError | null {
  if (isApiError(err)) return err;
  if (err instanceof Error) {
    return new ApiError({ status: 0, code: 'UNKNOWN_ERROR', message: err.message, cause: err });
  }
  return null;
}

export function formatFieldErrors(fieldErrors: Record<string, FieldError[]>): Record<string, string[]> {
  const out: Record<string, string[]> = {};
  for (const [field, errs] of Object.entries(fieldErrors)) {
    out[field] = errs.map(e => e.message);
  }
  return out;
}
