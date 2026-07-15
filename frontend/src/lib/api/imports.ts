/**
 * Import Jobs API — bulk student import via POST /students/import,
 * SSE progress streaming, and failure retrieval.
 *
 * Synced with backend:
 *   - backend/internal/students/domain.go (ImportRow, ImportRequest, ImportResponse)
 *   - backend/internal/imports/domain.go (Job, ProgressEvent, RowFailure, ImportJobStatus, ImportFailureType)
 *   - backend/internal/imports/handler.go (GET /imports/:job_id/stream)
 */

import { api, ApiError } from "./client";

// ============================================================================
// Import Job Status (backend: ImportJobStatus — 6-value enum)
// ============================================================================

export type ImportJobStatus =
    | "pending"
    | "processing"
    | "completed"
    | "completed_with_errors"
    | "failed"
    | "cancelling"
    | "cancelled";

// ============================================================================
// Import Failure Type (backend: ImportFailureType)
// ============================================================================

export type ImportFailureType =
    | "SCHEMA_VALIDATION"
    | "DATABASE_CONSTRAINT"
    | "BUSINESS_RULE_VIOLATION"
    | "INVALID_CLASS_REFERENCE"
    | "DB_CONSTRAINT_VIOLATION"
    | "DUPLICATE_ADMISSION_NUMBER"
    | "DUPLICATE_UPI_NUMBER"
    | "DUPLICATE_KNEC_NUMBER"
    // Bulk invitation failure types
    | "DUPLICATE_EMAIL"
    | "INVALID_EMAIL_FORMAT"
    | "STYTCH_API_ERROR"
    | "INVITATION_INSERT_FAILED";

// ============================================================================
// Import Row — a single student row in a bulk import
// Matches backend internal/students/domain.go ImportRow
// ============================================================================

export interface ImportRow {
    full_name: string;
    gender: "M" | "F";
    date_of_birth?: string | null;
    upi_number?: string | null;
    knec_assessment_number?: string | null;
    admission_number?: string | null;
    /** Class ID to enroll the student in. Empty/omit to create without enrollment. */
    class_id?: string;
}

// ============================================================================
// Import Request / Response
// Matches backend POST /students/import contract
// ============================================================================

export interface ImportRequest {
    idempotency_key?: string | null;
    rows: ImportRow[];
}

export interface ImportResponse {
    job_id: string;
    total_records: number;
    total_chunks: number;
    status: ImportJobStatus;
    /**
     * Indicates the response reflects a pre-existing job (idempotent replay)
     * rather than a newly created one. When true the HTTP status was 200
     * instead of 201.
     */
    is_replay?: boolean;
}

// ============================================================================
// Import Job (matches backend internal/imports/domain.go Job — flattened)
// ============================================================================

export interface ImportJob {
    id: string;
    tenant_id: string;
    school_id: string;
    job_type: "STAFF_INVITE" | "STUDENT_IMPORT";
    role?: string | null;
    created_by?: string | null;
    status: ImportJobStatus;
    total_records: number;
    processed_records: number;
    success_count: number;
    failed_count: number;
    idempotency_key?: string | null;
    total_chunks: number;
    processed_chunks: number;
    metadata?: Record<string, unknown>;
    created_at: string;
    started_at?: string | null;
    completed_at?: string | null;
    last_progress_at?: string | null;
}

// ============================================================================
// Progress Event — delivered over SSE
// Matches backend internal/imports/domain.go ProgressEvent
// ============================================================================

export interface ImportProgressEvent {
    job_id: string;
    status: ImportJobStatus;
    total_records: number;
    processed_records: number;
    success_count: number;
    failed_count: number;
    total_chunks: number;
    processed_chunks: number;
}

// ============================================================================
// Row Failure — a single failed row from the failures endpoint
// Matches backend internal/imports/domain.go RowFailure
// ============================================================================

export interface ImportRowFailure {
    row_number: number;
    raw_payload: Record<string, unknown>;
    error_message: string;
    error_type: ImportFailureType;
}

// ============================================================================
// Check Duplicates Types (backend POST /api/v1/students/check-duplicates)
// ============================================================================

export interface CheckDuplicatesRequest {
    admission_numbers?: string[];
    upi_numbers?: string[];
    knec_assessment_numbers?: string[];
}

export interface CheckDuplicatesResponse {
    existing_admission_numbers: string[];
    existing_upi_numbers: string[];
    existing_knec_assessment_numbers: string[];
}

// ============================================================================
// Active Import Job Check
// ============================================================================

/**
 * Response from GET /api/v1/schools/:school_id/imports/active.
 * If active is true, job contains the currently-active import job.
 */
export interface ActiveImportJobResponse {
    active: boolean;
    job: ImportJob | null;
}

/**
 * Type guard to check if an error is an import_already_in_progress response.
 * The backend returns this when a CreateJob request is rejected because
 * another job is already active for the same school.
 * Returns the active_job_id if matched, or null otherwise.
 */
export function getImportAlreadyInProgress(err: unknown): string | null {
    if (
        err instanceof ApiError &&
        err.code === "import_already_in_progress" &&
        err.extra?.active_job_id
    ) {
        return String(err.extra.active_job_id);
    }
    return null;
}

// ============================================================================
// API Functions
// ============================================================================

/**
 * GET /imports/active — proactively check if an import job is already running
 * for the current school before showing the import form.
 * The active school is resolved from the authenticated session on the backend,
 * so no school_id parameter is needed from the frontend.
 * Returns either the active job's state or a clear "no active job" response.
 */
export async function getActiveImportJob(): Promise<ActiveImportJobResponse> {
    return api.get<ActiveImportJobResponse>("/api/v1/imports/active");
}

/**
 * POST /students/import — submit a bulk student import job.
 * Accepts an optional `idempotency_key` for safe retry semantics.
 * When an idempotency_key is provided and the same payload was already
 * submitted, the response will have `is_replay: true` and HTTP 200
 * instead of 201.
 *
 * The idempotency_key should be generated with crypto.randomUUID() at
 * the point the submit action begins and reused across retries for the
 * same import action (not across distinct import flows).
 */
export async function submitStudentImport(body: ImportRequest): Promise<ImportResponse> {
    return api.post<ImportResponse>("/api/v1/students/import", body);
}

/**
 * GET /imports/{job_id}/failures — retrieve failure records for a completed job.
 * Paginated, tenant-scoped. Returns array of ImportRowFailure.
 */
export async function getImportFailures(
    jobId: string,
    params: { page?: number; limit?: number } = {}
): Promise<{ failures: ImportRowFailure[]; total: number }> {
    const searchParams = new URLSearchParams();
    if (params.page) searchParams.set("page", String(params.page));
    if (params.limit) searchParams.set("limit", String(params.limit));
    const qs = searchParams.toString();

    return api.get<{ failures: ImportRowFailure[]; total: number }>(
        `/api/v1/imports/${jobId}/failures?${qs}`
    );
}

/**
 * POST /students/check-duplicates — check which values already exist in the DB.
 * Returns only values that already exist for the current school.
 */
export async function checkDuplicates(
    body: CheckDuplicatesRequest
): Promise<CheckDuplicatesResponse> {
    return api.post<CheckDuplicatesResponse>("/api/v1/students/check-duplicates", body);
}

/**
 * GET /imports/{job_id} — retrieve current job state (for polling fallback).
 */
export async function getImportJob(jobId: string): Promise<ImportJob> {
    return api.get<ImportJob>(`/api/v1/imports/${jobId}`);
}

/**
 * POST /imports/{job_id}/cancel — request cancellation of a running import job.
 * The job must be in 'processing' state to be cancellable.
 * Returns the updated job with status 'cancelling' on success.
 * Throws ApiError with code "job_not_cancellable" if the job cannot be cancelled.
 */
export async function cancelImportJob(jobId: string): Promise<ImportJob> {
    return api.post<ImportJob>(`/api/v1/imports/${jobId}/cancel`);
}

/**
 * GET /imports — list paginated import jobs for the active school.
 * Returns jobs ordered by created_at descending.
 */
export interface ListJobsResponse {
    data: ImportJob[];
    total: number;
    page: number;
    limit: number;
}

export async function listJobs(
    params: { page?: number; limit?: number } = {}
): Promise<ListJobsResponse> {
    const searchParams = new URLSearchParams();
    if (params.page) searchParams.set("page", String(params.page));
    if (params.limit) searchParams.set("limit", String(params.limit));
    const qs = searchParams.toString();

    return api.get<ListJobsResponse>(`/api/v1/imports?${qs}`);
}
