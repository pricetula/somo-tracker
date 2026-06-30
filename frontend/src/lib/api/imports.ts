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
import type {
    ImportStaffRecord,
    StartImportRequest,
    StartImportResponse,
    ImportJob,
    TrackImportResponse,
    ImportProgressEvent,
    FailedInvitation,
    ListFailedInvitationsResponse,
} from "./generated";

// ─── Re-export generated types ───────────────────────────────────────────

export type {
    ImportStaffRecord,
    StartImportRequest,
    StartImportResponse,
    ImportJob,
    TrackImportResponse,
    ImportProgressEvent,
    FailedInvitation,
    ListFailedInvitationsResponse,
};

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
