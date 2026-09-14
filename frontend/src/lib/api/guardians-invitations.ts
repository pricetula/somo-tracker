import { api } from "./client";

export interface InvitationRow {
    email: string;
    full_name: string;
}

export interface BulkInvitationRequest {
    invitations: InvitationRow[];
}

export interface BulkInvitationResponse {
    job_id: string;
    status: string;
    total_records: number;
    message: string;
}

export interface InvitationJob {
    id: string;
    job_type: string;
    status: string;
    total_records: number;
    succeeded_count: number;
    failed_count: number;
    deferred_count: number;
}

export async function createGuardiansInvitations(
    payload: BulkInvitationRequest,
    idempotencyKey?: string
): Promise<BulkInvitationResponse> {
    return api.post<BulkInvitationResponse>("/api/guardians/invitations", payload, {
        headers: idempotencyKey ? { "Idempotency-Key": idempotencyKey } : {},
    });
}

export async function getGuardiansInvitationJob(jobId: string): Promise<InvitationJob> {
    return api.get<InvitationJob>(`/api/guardians/invitations/jobs/${jobId}`);
}

export async function retryGuardiansInvitations(
    jobId: string
): Promise<{ message: string; count: number }> {
    return api.post<{ message: string; count: number }>(
        `/api/guardians/invitations/jobs/${jobId}/retry-failed`,
        {}
    );
}

export function subscribeGuardiansInvitationEvents(
    jobId: string,
    onProgress: (data: Record<string, unknown>) => void,
    onHeartbeat: () => void = () => {},
    onError: (err: Error) => void = () => {}
): () => void {
    if (typeof window === "undefined") return () => {};

    const url = `/backend/api/guardians/invitations/jobs/${jobId}/events`;
    const evtSource = new EventSource(url);

    evtSource.addEventListener("progress", (e) => {
        try {
            onProgress(JSON.parse(e.data));
        } catch {}
    });

    evtSource.addEventListener("heartbeat", () => onHeartbeat());
    evtSource.addEventListener("error", () => onError(new Error("SSE error")));

    return () => evtSource.close();
}
