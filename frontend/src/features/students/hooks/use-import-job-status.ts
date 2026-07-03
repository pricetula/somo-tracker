/**
 * Hook that wraps an EventSource for import job SSE progress.
 *
 * Connects to GET /imports/{job_id}/stream, parses "state" and "progress"
 * events into a unified ImportJobStatus state.
 *
 * Native EventSource reconnects automatically on network blips.
 * This hook's onError is only for terminal-level failures (invalid job, etc.)
 * — transient network drops are handled by the browser's EventSource reconnect.
 */

"use client";

import * as React from "react";
import {
    buildImportStreamUrl,
    type ImportJobStatus,
    type ImportProgressEvent,
} from "@/lib/api/imports";

// ─── State shape ──────────────────────────────────────────────────────────

export interface ImportJobState {
    status: ImportJobStatus;
    totalRecords: number;
    processedRecords: number;
    successCount: number;
    failedCount: number;
    totalChunks: number;
    processedChunks: number;
}

const INITIAL_STATE: ImportJobState = {
    status: "pending",
    totalRecords: 0,
    processedRecords: 0,
    successCount: 0,
    failedCount: 0,
    totalChunks: 0,
    processedChunks: 0,
};

// ─── Hook ─────────────────────────────────────────────────────────────────

/**
 * Subscribe to SSE progress for an import job.
 *
 * @param jobId - The import job ID to subscribe to. Null/undefined disables.
 * @param onTerminal - Callback fired when the job reaches a terminal state.
 *
 * Returns ImportJobState reflecting the latest job progress.
 */
export function useImportJobStatus(
    jobId: string | null | undefined,
    onTerminal?: (state: ImportJobState) => void
): ImportJobState {
    const [state, setState] = React.useState<ImportJobState>(INITIAL_STATE);
    const onTerminalRef = React.useRef(onTerminal);

    // Sync ref via effect, not during render
    React.useEffect(() => {
        onTerminalRef.current = onTerminal;
    }, [onTerminal]);

    // Track previous jobId to detect transitions
    const prevJobIdRef = React.useRef(jobId);
    React.useEffect(() => {
        prevJobIdRef.current = jobId;
    });

    // Set up SSE when jobId is provided; skip when falsy
    React.useEffect(() => {
        if (!jobId) return;

        const url = buildImportStreamUrl(jobId);
        let eventSource: EventSource | null = null;
        let closed = false;

        function openEventSource() {
            if (closed) return;

            eventSource = new EventSource(url);

            eventSource.addEventListener("state", (e: MessageEvent) => {
                try {
                    const data: ImportProgressEvent = JSON.parse(e.data);
                    updateState(data);
                } catch {
                    // Ignore malformed events
                }
            });

            eventSource.addEventListener("progress", (e: MessageEvent) => {
                try {
                    const data: ImportProgressEvent = JSON.parse(e.data);
                    updateState(data);
                } catch {
                    // Ignore malformed events
                }
            });

            eventSource.addEventListener("error", () => {
                // EventSource fires "error" on transient connection drops.
                // The browser natively reconnects, so we don't treat this as
                // a terminal failure. Only act if readyState is CLOSED.
                if (eventSource?.readyState === EventSource.CLOSED) {
                    // Connection permanently lost — do not close again in cleanup
                    closed = true;
                }
            });
        }

        function updateState(data: ImportProgressEvent) {
            setState((prev) => {
                const next: ImportJobState = {
                    status: data.status as ImportJobStatus,
                    totalRecords: data.total_records ?? prev.totalRecords,
                    processedRecords: data.processed_records ?? prev.processedRecords,
                    successCount: data.success_count ?? prev.successCount,
                    failedCount: data.failed_count ?? prev.failedCount,
                    totalChunks: data.total_chunks ?? prev.totalChunks,
                    processedChunks: data.processed_chunks ?? prev.processedChunks,
                };

                // Fire terminal callback if job is done
                const terminal =
                    next.status === "completed" ||
                    next.status === "completed_with_errors" ||
                    next.status === "failed" ||
                    next.status === "cancelled";

                if (terminal) {
                    // Defer to avoid setState-during-render issues
                    setTimeout(() => onTerminalRef.current?.(next), 0);
                }

                return next;
            });
        }

        openEventSource();

        return () => {
            closed = true;
            eventSource?.close();
            eventSource = null;
        };
    }, [jobId]);

    // When no jobId, skip SSE but keep initial state
    return state;
}
