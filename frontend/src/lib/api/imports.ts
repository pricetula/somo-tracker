/**
 * Staff Import API functions.
 *
 * Endpoints:
 *   POST /api/v1/imports/staff    — start a bulk import job
 *   GET  /api/v1/imports/staff/track/:id        — poll job status
 *   GET  /api/v1/imports/staff/track/:id/sse    — SSE real-time progress
 *   GET  /api/v1/imports/staff/:id/failures     — list failed invitations
 */

import { api } from "./client";

// ─── Types ─────────────────────────────────────────────────────────────────

export interface ImportStaffRecord {
    temp_id: string;
    email: string;
    first_name: string;
    last_name: string;
    phone?: string;
    registration_number?: string;
}

export interface StartImportRequest {
    role: "SCHOOL_ADMIN" | "NURSE" | "FINANCE" | "TEACHER";
    records: ImportStaffRecord[];
}

export interface StartImportResponse {
    import_job_id: string;
    status: string;
    total: number;
}

export interface ImportJob {
    id: string;
    tenant_id: string;
    school_id: string;
    role: string;
    created_by?: string;
    status: string;
    total_records: number;
    processed_records: number;
    success_count: number;
    failed_count: number;
    parent_import_job_id?: string;
    created_at: string;
    started_at?: string;
    completed_at?: string;
}

export interface TrackImportResponse {
    job: ImportJob;
    failed_records: number;
}

export interface ImportProgressEvent {
    type: "connected" | "import_progress" | "import_finished" | "import_error";
    import_job_id: string;
    status?: string;
    processed_records?: number;
    success_count?: number;
    failed_count?: number;
    total_records?: number;
}

export interface FailedInvitation {
    id: string;
    email: string;
    first_name?: string;
    last_name?: string;
    phone?: string;
    error_message?: string;
}

export interface ListFailedInvitationsResponse {
    invitations: FailedInvitation[];
}

// ─── API Functions ─────────────────────────────────────────────────────────

/** Start a bulk staff import job. Returns 202 Accepted with job ID. */
export async function startImport(payload: StartImportRequest): Promise<StartImportResponse> {
    return api.post<StartImportResponse>("/api/v1/imports/staff", payload);
}

/** Poll the current status of an import job. */
export async function trackImport(jobID: string): Promise<TrackImportResponse> {
    return api.get<TrackImportResponse>(`/api/v1/imports/staff/track/${jobID}`);
}

/** Get failed invitations for a completed import job (recovery). */
export async function listFailedInvitations(jobID: string): Promise<ListFailedInvitationsResponse> {
    return api.get<ListFailedInvitationsResponse>(`/api/v1/imports/staff/${jobID}/failures`);
}

/** Create an SSE connection for real-time import progress. */
export function createImportSSE(jobID: string): EventSource {
    const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "";
    return new EventSource(`${API_BASE}/api/v1/imports/staff/track/${jobID}/sse`);
}
