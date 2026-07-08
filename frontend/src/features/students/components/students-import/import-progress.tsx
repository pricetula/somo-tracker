/**
 * ImportProgress — shared progress/result component for student imports.
 *
 * Polls GET /api/v1/imports/:job_id until terminal, fetches failures,
 * and displays progress bar, counts, and error details.
 *
 * Supports cancelling an in-progress import via the "Cancel Import" button,
 * visible only while the job status is 'processing'. After requesting
 * cancellation, the button is disabled and the polling loop picks up the
 * 'cancelling' -> 'cancelled' transition naturally.
 *
 * Used by both Manual Import and File Import flows.
 */

"use client";

import * as React from "react";
import { CheckCircle2, XCircle, Loader2, AlertTriangle, Ban } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { toast } from "sonner";

import {
    getImportJob,
    getImportFailures,
    checkDuplicates,
    cancelImportJob,
} from "@/lib/api/imports";
import type { ImportJob, ImportJobStatus, ImportRowFailure } from "@/lib/api/imports";

// ─── Constants ────────────────────────────────────────────────────────────

const TERMINAL_STATUSES: ImportJobStatus[] = [
    "completed",
    "completed_with_errors",
    "failed",
    "cancelled",
];

// Polling backoff schedule: fast while the job is expected to be quick,
// then progressively lighter for long-running imports.
// - 0–30s:     every 1.5s  (current default, unchanged)
// - 30s–2min:  every 3s
// - beyond:    every 10s   (ceiling, never grows unbounded)
const POLL_INTERVAL_FAST = 1_500; // 0–30s
const POLL_INTERVAL_MEDIUM = 3_000; // 30s–2min
const POLL_INTERVAL_SLOW = 10_000; // beyond 2min
const ELAPSED_MEDIUM_MS = 30_000; // 30 seconds
const ELAPSED_SLOW_MS = 120_000; // 2 minutes

// Stalled-job threshold: if last_progress_at is older than this while the
// job is still processing/cancelling, show a non-alarming inline message.
const STALLED_THRESHOLD_MS = 120_000; // 2 minutes

// ─── Props ────────────────────────────────────────────────────────────────

interface ImportProgressProps {
    jobId: string;
    totalRecords: number;
    onDone: () => void;
    onRetry?: (failedPayloads: Record<string, unknown>[]) => void;
    /**
     * Fired when the polling loop first detects a terminal job status.
     * Used by the parent to clean up IndexedDB sessions (G2).
     * Will fire exactly once per mount, regardless of which button the
     * user clicks afterward, so an abandoned tab also gets cleaned up.
     */
    onTerminalStatus?: () => void;
}

// ─── Component ────────────────────────────────────────────────────────────

export function ImportProgress({
    jobId,
    totalRecords,
    onDone,
    onRetry,
    onTerminalStatus,
}: ImportProgressProps) {
    const [job, setJob] = React.useState<ImportJob | null>(null);
    const [failures, setFailures] = React.useState<ImportRowFailure[]>([]);
    const [cancelling, setCancelling] = React.useState(false);
    const pollRef = React.useRef<ReturnType<typeof setInterval> | null>(null);

    // ── Polling backoff ──────────────────────────────────────────────────
    // Calculate the appropriate poll interval based on elapsed time since
    // the component mounted (≈ time since the job was submitted).
    // Lazy initializer is called once and avoids an impure Date.now() in render.
    const [mountTime] = React.useState(() => Date.now());

    const [elapsedMs, setElapsedMs] = React.useState(0);
    const [isStalled, setIsStalled] = React.useState(false);

    // Update elapsed time every second so the poll interval can react to
    // crossing the backoff thresholds.
    React.useEffect(() => {
        const timer = setInterval(() => {
            setElapsedMs(Date.now() - mountTime);
        }, 1000);
        return () => clearInterval(timer);
    }, [mountTime]);

    const pollInterval = React.useMemo(() => {
        if (elapsedMs < ELAPSED_MEDIUM_MS) return POLL_INTERVAL_FAST;
        if (elapsedMs < ELAPSED_SLOW_MS) return POLL_INTERVAL_MEDIUM;
        return POLL_INTERVAL_SLOW;
    }, [elapsedMs]);

    // Track whether onTerminalStatus has been called (exactly once)
    const terminalFired = React.useRef(false);

    // Cleanup polling on unmount
    React.useEffect(() => {
        return () => {
            if (pollRef.current) clearInterval(pollRef.current);
        };
    }, []);

    // Start polling on mount. The interval changes reactively via pollRef.
    React.useEffect(() => {
        const poll = async () => {
            try {
                const current = await getImportJob(jobId);
                setJob(current);

                // Stalled-job detection: computed inside effect where
                // Date.now() is acceptable (not during render).
                if (
                    !TERMINAL_STATUSES.includes(current.status) &&
                    (current.status === "processing" || current.status === "cancelling") &&
                    current.last_progress_at
                ) {
                    const elapsed = Date.now() - new Date(current.last_progress_at).getTime();
                    setIsStalled(elapsed > STALLED_THRESHOLD_MS);
                } else {
                    setIsStalled(false);
                }

                if (TERMINAL_STATUSES.includes(current.status)) {
                    if (pollRef.current) {
                        clearInterval(pollRef.current);
                        pollRef.current = null;
                    }

                    // G2: Fire terminal-status callback exactly once
                    if (!terminalFired.current) {
                        terminalFired.current = true;
                        onTerminalStatus?.();
                    }

                    // Fetch failures if the job completed with errors
                    if (current.status === "completed_with_errors" || current.status === "failed") {
                        try {
                            const failResult = await getImportFailures(jobId, { limit: 200 });
                            setFailures(failResult.failures);
                        } catch (fetchErr) {
                            console.error("Failed to fetch import failures:", fetchErr);
                        }
                    }
                }
            } catch {
                // Transient — just wait for next tick
            }
        };

        poll();
        pollRef.current = setInterval(poll, pollInterval);

        return () => {
            if (pollRef.current) clearInterval(pollRef.current);
        };
    }, [jobId, onTerminalStatus, pollInterval, setIsStalled]);

    // Derive state
    const isTerminal = job && TERMINAL_STATUSES.includes(job.status);
    const isStreaming = job && !isTerminal;
    const allSucceeded = job?.status === "completed";
    const isCancelled = job?.status === "cancelled";
    const isCancelling = job?.status === "cancelling";

    const processed = job ? job.success_count + job.failed_count : 0;
    const percentComplete =
        totalRecords > 0 ? Math.min(100, Math.round((processed / totalRecords) * 100)) : 0;

    const retryPayloads = React.useMemo(
        () =>
            failures
                .map((f) => f.raw_payload)
                .filter((p): p is Record<string, unknown> => p !== null && typeof p === "object"),
        [failures]
    );

    const [retrying, setRetrying] = React.useState(false);

    const handleRetry = React.useCallback(async () => {
        if (!onRetry || retrying) return;

        setRetrying(true);
        try {
            // Check for existing-record conflicts before retrying
            const admNumbers = retryPayloads
                .map((p) => (p.admission_number as string | undefined) ?? "")
                .filter(Boolean);
            const upiNumbers = retryPayloads
                .map((p) => (p.upi_number as string | undefined) ?? "")
                .filter(Boolean);
            const knecNumbers = retryPayloads
                .map((p) => (p.knec_assessment_number as string | undefined) ?? "")
                .filter(Boolean);

            if (admNumbers.length > 0 || upiNumbers.length > 0 || knecNumbers.length > 0) {
                try {
                    const result = await checkDuplicates({
                        admission_numbers: admNumbers,
                        upi_numbers: upiNumbers,
                        knec_assessment_numbers: knecNumbers,
                    });
                    const hasConflicts =
                        result.existing_admission_numbers.length > 0 ||
                        result.existing_upi_numbers.length > 0 ||
                        result.existing_knec_assessment_numbers.length > 0;

                    if (hasConflicts) {
                        const conflictMsgs: string[] = [];
                        if (result.existing_admission_numbers.length > 0) {
                            conflictMsgs.push(
                                `Admission number(s) ${result.existing_admission_numbers.join(", ")} already exist`
                            );
                        }
                        if (result.existing_upi_numbers.length > 0) {
                            conflictMsgs.push(
                                `UPI number(s) ${result.existing_upi_numbers.join(", ")} already exist`
                            );
                        }
                        if (result.existing_knec_assessment_numbers.length > 0) {
                            conflictMsgs.push(
                                `KNEC number(s) ${result.existing_knec_assessment_numbers.join(", ")} already exist`
                            );
                        }
                        toast.error(
                            `Cannot retry — some values now conflict with existing records:\n${conflictMsgs.join("\n")}`
                        );
                        setRetrying(false);
                        return;
                    }
                } catch {
                    // If the check itself fails, allow retry to proceed
                    console.warn("Duplicate check failed during retry, proceeding anyway");
                }
            }

            onRetry(retryPayloads);
        } finally {
            setRetrying(false);
        }
    }, [retryPayloads, onRetry, retrying]);

    // ─── Cancel handler ─────────────────────────────────────────────────
    const handleCancel = React.useCallback(async () => {
        if (cancelling) return; // prevent double-clicks

        setCancelling(true);
        try {
            await cancelImportJob(jobId);
            // The polling loop will pick up 'cancelling' -> 'cancelled'
            // transition naturally — no special handling needed here.
        } catch {
            setCancelling(false);
            toast.error("Failed to cancel import. Please try again.");
        }
    }, [jobId, cancelling]);

    return (
        <div className="space-y-4">
            {/* Progress bar */}
            <div className="space-y-2">
                <div className="flex items-center justify-between text-sm">
                    <span className="text-muted-foreground">
                        {!job && "Starting…"}
                        {isCancelling && "Cancelling…"}
                        {isStreaming && !isCancelling && "Importing…"}
                        {isTerminal && !isCancelled && "Complete"}
                        {isCancelled && "Cancelled"}
                    </span>
                    <span className="font-medium">
                        {processed} / {totalRecords}
                    </span>
                </div>

                {job && (
                    <>
                        <Progress value={percentComplete} className="h-2" />

                        <div className="flex items-center gap-4 text-xs">
                            {job.success_count > 0 && (
                                <span className="flex items-center gap-1 text-emerald-600">
                                    <CheckCircle2 className="size-3.5" />
                                    {job.success_count} succeeded
                                </span>
                            )}
                            {job.failed_count > 0 && (
                                <span className="text-destructive flex items-center gap-1">
                                    <XCircle className="size-3.5" />
                                    {job.failed_count} failed
                                </span>
                            )}
                            {isStreaming && !isCancelling && (
                                <span className="text-muted-foreground flex items-center gap-1">
                                    <Loader2 className="size-3.5 animate-spin" />
                                    Processing chunk {job.processed_chunks} of {job.total_chunks}
                                </span>
                            )}
                            {isCancelling && (
                                <span className="text-muted-foreground flex items-center gap-1">
                                    <Ban className="size-3.5" />
                                    Waiting for in-flight chunks to finish…
                                </span>
                            )}
                        </div>
                    </>
                )}
            </div>

            {/* Stalled-job message — visible while processing/cancelling with no recent progress */}
            {isStalled && (
                <div className="flex items-start gap-2 rounded-md bg-amber-50 px-3 py-2 text-xs text-amber-700">
                    <AlertTriangle className="mt-0.5 size-3.5 shrink-0" />
                    <span>
                        This import is taking longer than usual — you can keep waiting or cancel it.
                    </span>
                </div>
            )}

            {/* Cancel button — visible only while status is 'processing' */}
            {job?.status === "processing" && (
                <div className="flex justify-end">
                    <Button
                        variant="outline"
                        size="sm"
                        onClick={handleCancel}
                        disabled={cancelling}
                        className="text-destructive hover:text-destructive"
                    >
                        {cancelling ? (
                            <>
                                <Loader2 className="mr-1.5 size-3.5 animate-spin" />
                                Cancelling…
                            </>
                        ) : (
                            <>
                                <Ban className="mr-1.5 size-3.5" />
                                Cancel Import
                            </>
                        )}
                    </Button>
                </div>
            )}

            {/* Result banner */}
            {isTerminal && (
                <div
                    className={`flex items-start gap-3 rounded-md px-4 py-3 text-sm ${
                        allSucceeded
                            ? "bg-emerald-50 text-emerald-800"
                            : isCancelled
                              ? "bg-slate-50 text-slate-800"
                              : "bg-amber-50 text-amber-800"
                    }`}
                >
                    {allSucceeded ? (
                        <CheckCircle2 className="mt-0.5 size-4 shrink-0 text-emerald-600" />
                    ) : isCancelled ? (
                        <Ban className="mt-0.5 size-4 shrink-0 text-slate-600" />
                    ) : (
                        <AlertTriangle className="mt-0.5 size-4 shrink-0 text-amber-600" />
                    )}
                    <div className="flex-1">
                        <p className="font-medium">
                            {allSucceeded
                                ? "Import completed successfully"
                                : isCancelled
                                  ? "Import cancelled"
                                  : "Import completed with issues"}
                        </p>
                        {isCancelled ? (
                            <p className="mt-0.5 text-xs opacity-80">
                                {job!.success_count > 0
                                    ? `${job!.success_count} student${job!.success_count === 1 ? "" : "s"} ${
                                          job!.success_count === 1 ? "was" : "were"
                                      } already added before cancellation took effect.`
                                    : "No students were added."}
                                {job!.failed_count > 0 &&
                                    ` ${job!.failed_count} row${job!.failed_count === 1 ? "" : "s"} failed before cancellation.`}
                            </p>
                        ) : (
                            <p className="mt-0.5 text-xs opacity-80">
                                {job!.success_count} succeeded, {job!.failed_count} failed
                                {job!.total_records > 0 && ` out of ${job!.total_records} total`}
                            </p>
                        )}

                        {/* Failure details (only for completed_with_errors, not cancelled) */}
                        {!isCancelled && failures.length > 0 && (
                            <div className="mt-2 space-y-1">
                                <p className="text-xs font-medium">Failed rows:</p>
                                <ul className="list-inside list-disc space-y-0.5 text-xs opacity-80">
                                    {failures.slice(0, 20).map((f, i) => (
                                        <li key={i}>
                                            Row {f.row_number}: {f.error_message}
                                        </li>
                                    ))}
                                    {failures.length > 20 && (
                                        <li className="list-none text-xs italic">
                                            …and {failures.length - 20} more
                                        </li>
                                    )}
                                </ul>
                            </div>
                        )}

                        {/* Actions */}
                        <div className="mt-2 flex items-center gap-2">
                            <Button variant="outline" size="sm" onClick={onDone}>
                                Done
                            </Button>
                            {!isCancelled && retryPayloads.length > 0 && onRetry && (
                                <Button
                                    variant="outline"
                                    size="sm"
                                    onClick={handleRetry}
                                    disabled={retrying}
                                >
                                    {retrying ? "Checking…" : "Retry failed"}
                                </Button>
                            )}
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
}
