/**
 * ImportProgress — shared progress/result component for student imports.
 *
 * Polls GET /api/v1/imports/:job_id until terminal, fetches failures,
 * and displays progress bar, counts, and error details.
 *
 * Used by both Manual Import and File Import flows.
 */

"use client";

import * as React from "react";
import { CheckCircle2, XCircle, Loader2, AlertTriangle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { toast } from "sonner";

import { getImportJob, getImportFailures, checkDuplicates } from "@/lib/api/imports";
import type { ImportJob, ImportJobStatus, ImportRowFailure } from "@/lib/api/imports";

// ─── Constants ────────────────────────────────────────────────────────────

const TERMINAL_STATUSES: ImportJobStatus[] = [
    "completed",
    "completed_with_errors",
    "failed",
    "cancelled",
];

const POLL_INTERVAL_MS = 1500;

// ─── Props ────────────────────────────────────────────────────────────────

interface ImportProgressProps {
    jobId: string;
    totalRecords: number;
    onDone: () => void;
    onRetry?: (failedPayloads: Record<string, unknown>[]) => void;
}

// ─── Component ────────────────────────────────────────────────────────────

export function ImportProgress({ jobId, totalRecords, onDone, onRetry }: ImportProgressProps) {
    const [job, setJob] = React.useState<ImportJob | null>(null);
    const [failures, setFailures] = React.useState<ImportRowFailure[]>([]);
    const pollRef = React.useRef<ReturnType<typeof setInterval> | null>(null);

    // Cleanup polling on unmount
    React.useEffect(() => {
        return () => {
            if (pollRef.current) clearInterval(pollRef.current);
        };
    }, []);

    // Start polling on mount
    React.useEffect(() => {
        const poll = async () => {
            try {
                const current = await getImportJob(jobId);
                setJob(current);

                if (TERMINAL_STATUSES.includes(current.status)) {
                    if (pollRef.current) {
                        clearInterval(pollRef.current);
                        pollRef.current = null;
                    }

                    // Fetch failures if any
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
        pollRef.current = setInterval(poll, POLL_INTERVAL_MS);

        return () => {
            if (pollRef.current) clearInterval(pollRef.current);
        };
    }, [jobId]);

    // Derive state
    const isTerminal = job && TERMINAL_STATUSES.includes(job.status);
    const isStreaming = job && !isTerminal;
    const allSucceeded = job?.status === "completed";

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

    return (
        <div className="space-y-4">
            {/* Progress bar */}
            <div className="space-y-2">
                <div className="flex items-center justify-between text-sm">
                    <span className="text-muted-foreground">
                        {!job && "Starting…"}
                        {isStreaming && "Importing…"}
                        {isTerminal && "Complete"}
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
                            {isStreaming && (
                                <span className="text-muted-foreground flex items-center gap-1">
                                    <Loader2 className="size-3.5 animate-spin" />
                                    Processing chunk {job.processed_chunks} of {job.total_chunks}
                                </span>
                            )}
                        </div>
                    </>
                )}
            </div>

            {/* Result banner */}
            {isTerminal && (
                <div
                    className={`flex items-start gap-3 rounded-md px-4 py-3 text-sm ${
                        allSucceeded
                            ? "bg-emerald-50 text-emerald-800"
                            : "bg-amber-50 text-amber-800"
                    }`}
                >
                    {allSucceeded ? (
                        <CheckCircle2 className="mt-0.5 size-4 shrink-0 text-emerald-600" />
                    ) : (
                        <AlertTriangle className="mt-0.5 size-4 shrink-0 text-amber-600" />
                    )}
                    <div className="flex-1">
                        <p className="font-medium">
                            {allSucceeded
                                ? "Import completed successfully"
                                : "Import completed with issues"}
                        </p>
                        <p className="mt-0.5 text-xs opacity-80">
                            {job!.success_count} succeeded, {job!.failed_count} failed
                            {job!.total_records > 0 && ` out of ${job!.total_records} total`}
                        </p>

                        {/* Failure details */}
                        {failures.length > 0 && (
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
                            {retryPayloads.length > 0 && onRetry && (
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
