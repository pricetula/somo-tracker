/**
 * Import Jobs API — bulk student import via POST /students/import,
 * SSE progress streaming, and failure retrieval.
 *
 * Synced with backend:
 *   - backend/internal/students/domain.go (ImportRow, ImportRequest, ImportResponse)
 *   - backend/internal/imports/domain.go (Job, ProgressEvent, RowFailure, ImportJobStatus, ImportFailureType)
 *   - backend/internal/imports/handler.go (GET /imports/:job_id/stream)
 */

import { api } from "./client";

// ============================================================================
// Import Job Status (backend: ImportJobStatus — 6-value enum)
// ============================================================================

export type ImportJobStatus =
    | "pending"
    | "processing"
    | "completed"
    | "completed_with_errors"
    | "failed"
    | "cancelled";

// ============================================================================
// Import Failure Type (backend: ImportFailureType)
// ============================================================================

export type ImportFailureType =
    | "SCHEMA_VALIDATION"
    | "DATABASE_CONSTRAINT"
    | "BUSINESS_RULE_VIOLATION";

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
// API Functions
// ============================================================================

/**
 * POST /students/import — submit a bulk student import job.
 * Returns job_id, total_records, total_chunks, and initial status.
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
 * GET /imports/{job_id} — retrieve current job state (for polling fallback).
 */
export async function getImportJob(jobId: string): Promise<ImportJob> {
    return api.get<ImportJob>(`/api/v1/imports/${jobId}`);
}
