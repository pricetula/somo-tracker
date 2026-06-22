/**
 * TanStack Query hooks + RxJS SSE stream for the bulk staff import pipeline.
 */

"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Observable } from "rxjs";
import { toast } from "sonner";

import { getErrorMessage } from "@/lib/errors";

import {
    startImport,
    trackImport,
    listFailedInvitations,
    createImportSSE,
    type StartImportRequest,
    type StartImportResponse,
    type ImportProgressEvent,
    type TrackImportResponse,
    type ListFailedInvitationsResponse,
} from "@/lib/api/imports";

// ─── Query keys ─────────────────────────────────────────────────────────

export const importKeys = {
    all: ["imports"] as const,
    track: (jobID: string) => ["imports", "track", jobID] as const,
    failures: (jobID: string) => ["imports", "failures", jobID] as const,
};

// ─── Hooks ───────────────────────────────────────────────────────────────

/** Start a new bulk import job. Invalidates tracking queries on completion. */
export function useStartImport() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: StartImportRequest) => startImport(payload),
        onSuccess: (result: StartImportResponse) => {
            queryClient.setQueryData(importKeys.track(result.import_job_id), result);
            toast.success("Import started", {
                description: `Processing ${result.total} records...`,
            });
        },
        onError: (err) => {
            toast.error("Failed to start import", {
                description: getErrorMessage(err),
            });
        },
    });
}

/** Poll the status of an active import job. */
export function useTrackImport(jobID: string | null, opts: { enabled?: boolean } = {}) {
    const { enabled = true } = opts;

    return useQuery<TrackImportResponse>({
        queryKey: importKeys.track(jobID ?? ""),
        queryFn: () => trackImport(jobID!),
        enabled: !!jobID && enabled,
        refetchInterval: (query) => {
            const data = query.state.data;
            if (!data) return 3000;
            // Stop polling when done
            const status = data.job.status;
            if (status === "completed" || status === "completed_with_errors") return false;
            return 3000;
        },
        placeholderData: (prev) => prev,
    });
}

/** Fetch failed invitations for a completed import job. */
export function useImportFailures(jobID: string | null) {
    return useQuery<ListFailedInvitationsResponse>({
        queryKey: importKeys.failures(jobID ?? ""),
        queryFn: () => listFailedInvitations(jobID!),
        enabled: !!jobID,
    });
}

// ─── RxJS SSE Observable ────────────────────────────────────────────────

/**
 * Creates an RxJS Observable that streams ImportProgressEvent from an SSE
 * endpoint. If the SSE connection drops, falls back to polling.
 */
export function createImportProgressStream(jobID: string): Observable<ImportProgressEvent> {
    return new Observable<ImportProgressEvent>((subscriber) => {
        const eventSource = createImportSSE(jobID);

        eventSource.onmessage = (event: MessageEvent) => {
            try {
                const data: ImportProgressEvent = JSON.parse(event.data);
                subscriber.next(data);

                if (data.type === "import_finished") {
                    subscriber.complete();
                    eventSource.close();
                }
            } catch (err) {
                subscriber.error(err);
            }
        };

        eventSource.onerror = () => {
            // SSE connection dropped — fall back to polling
            eventSource.close();
            pollFallback(jobID, subscriber);
        };

        return () => {
            eventSource.close();
        };
    });
}

/** Polling fallback when SSE connection drops. */
function pollFallback(
    jobID: string,
    subscriber: {
        next: (event: ImportProgressEvent) => void;
        complete: () => void;
        error: (err: unknown) => void;
    }
) {
    const interval = setInterval(async () => {
        try {
            const result = await trackImport(jobID);
            const event: ImportProgressEvent = {
                type: "import_progress",
                import_job_id: jobID,
                status: result.job.status,
                processed_records: result.job.processed_records,
                success_count: result.job.success_count,
                failed_count: result.job.failed_count,
                total_records: result.job.total_records,
            };
            subscriber.next(event);

            if (
                result.job.status === "completed" ||
                result.job.status === "completed_with_errors"
            ) {
                const finished: ImportProgressEvent = {
                    type: "import_finished",
                    import_job_id: jobID,
                    status: result.job.status,
                    processed_records: result.job.processed_records,
                    success_count: result.job.success_count,
                    failed_count: result.job.failed_count,
                    total_records: result.job.total_records,
                };
                subscriber.next(finished);
                subscriber.complete();
                clearInterval(interval);
            }
        } catch (err) {
            subscriber.error(err);
            clearInterval(interval);
        }
    }, 3000);

    return () => clearInterval(interval);
}
