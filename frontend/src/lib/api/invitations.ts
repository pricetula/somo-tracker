/**
 * Admin Invitations API — bulk member invitation ingestion.
 *
 * Backend contract (backend/internal/api/admin_invitation_handler.go):
 *
 *   POST /admins/invitations
 *   GET  /admins/invitations/jobs/:job_id
 *   POST /admins/invitations/jobs/:job_id/retry-failed
 *   GET  /admins/invitations/jobs/:job_id/events (SSE)
 */

import { api } from "./client";
import type {
    BulkInvitationRequest,
    BulkInvitationResponse,
    InvitationJob,
    InvitationRetryResponse,
    InvitationRow,
} from "./generated";

// ─── Types ────────────────────────────────────────────────────────────────

export interface CreateInvitationsPayload {
    invitations: InvitationRow[];
    idempotencyKey?: string;
}

// ─── Bulk invite ──────────────────────────────────────────────────────────

export async function createInvitations(
    payload: CreateInvitationsPayload
): Promise<BulkInvitationResponse> {
    const body: BulkInvitationRequest = {
        invitations: payload.invitations,
    };
    return api.post<BulkInvitationResponse>("/admins/invitations", body, {
        headers: payload.idempotencyKey ? { "Idempotency-Key": payload.idempotencyKey } : {},
    });
}

// ─── Job status ───────────────────────────────────────────────────────────

export async function getInvitationJob(jobId: string): Promise<InvitationJob> {
    return api.get<InvitationJob>(`/admins/invitations/jobs/${jobId}`);
}

// ─── Retry failed ─────────────────────────────────────────────────────────

export async function retryFailedInvitations(jobId: string): Promise<InvitationRetryResponse> {
    return api.post<InvitationRetryResponse>(`/admins/invitations/jobs/${jobId}/retry-failed`, {});
}

// ─── SSE events stream ────────────────────────────────────────────────────

/** Subscribe to progress events for a bulk invitation job. */
export function subscribeInvitationEvents(
    jobId: string,
    onProgress: (data: Record<string, unknown>) => void,
    onHeartbeat: () => void = () => {},
    onError: (err: Error) => void = () => {}
): () => void {
    if (typeof window === "undefined") {
        // Server context: return no-op cleanup
        return () => {};
    }

    const url = `/admins/invitations/jobs/${jobId}/events`;
    const evtSource = new EventSource(url);

    evtSource.addEventListener("progress", (e) => {
        try {
            onProgress(JSON.parse(e.data));
        } catch {
            // ignore malformed payload
        }
    });

    evtSource.addEventListener("heartbeat", () => {
        onHeartbeat();
    });

    evtSource.addEventListener("error", () => {
        onError(new Error("invitation SSE stream error"));
    });

    return () => {
        evtSource.close();
    };
}
