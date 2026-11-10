export interface FieldError {
  code: string;
  message: string;
}

export interface ApiErrorInit {
  status: number;
  code: string;
  message: string;
  fieldErrors?: Record<string, FieldError[]>;
  requestId?: string;
  retryAfterSeconds?: number;
  cause?: unknown;
}

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly fieldErrors: Record<string, FieldError[]>;
  readonly requestId?: string;
  readonly retryAfterSeconds?: number;

  constructor(init: ApiErrorInit) {
    super(init.message);
    this.name = 'ApiError';
    this.status = init.status;
    this.code = init.code;
    this.fieldErrors = init.fieldErrors ?? {};
    this.requestId = init.requestId;
    this.retryAfterSeconds = init.retryAfterSeconds;
    // Maintains proper stack trace
    if (Error.captureStackTrace) {
      Error.captureStackTrace(this, ApiError);
    }
  }
}

export function isApiError(e: unknown): e is ApiError {
  return e instanceof ApiError ||
    (typeof e === 'object' && e !== null &&
      (e as { name?: unknown }).name === 'ApiError' &&
      typeof (e as { status?: unknown }).status === 'number');
}

export function codeFromStatus(status: number): string {
  switch (status) {
    case 400: return 'BAD_REQUEST';
    case 401: return 'UNAUTHENTICATED';
    case 403: return 'FORBIDDEN';
    case 404: return 'NOT_FOUND';
    case 405: return 'METHOD_NOT_ALLOWED';
    case 408: return 'REQUEST_TIMEOUT';
    case 409: return 'CONFLICT';
    case 413: return 'PAYLOAD_TOO_LARGE';
    case 415: return 'UNSUPPORTED_MEDIA_TYPE';
    case 422: return 'VALIDATION_FAILED';
    case 429: return 'RATE_LIMITED';
    case 500: return 'INTERNAL_ERROR';
    case 502: return 'BAD_GATEWAY';
    case 503: return 'SERVICE_UNAVAILABLE';
    case 504: return 'GATEWAY_TIMEOUT';
    default:
      return status >= 500 ? 'INTERNAL_ERROR' : 'CLIENT_ERROR';
  }
}

export function parseRetryAfter(header: string | null): number | undefined {
  if (!header) return undefined;
  const s = parseInt(header, 10);
  return Number.isNaN(s) ? undefined : s;
}
