/**
 * Stage 4: Submitting & Progress via SSE.
 *
 * - Dispatched the POST, now watching via EventSource
 * - Shows progress bar via processed_records / total_records
 * - On completed: clears IndexedDB, success toast
 * - On completed_with_errors: fetches failures, reconciles back to rows
 * - On failed: reverts to READY
 * - On resume: reconnects to existing job_id
 */

"use client";

import * as React from "react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { Badge } from "@/components/ui/badge";
import { Loader2 } from "lucide-react";

import { useImportStore } from "../hooks/use-import-store";
import { useImportJobStatus, type ImportJobState } from "../hooks/use-import-job-status";
import { getImportFailures } from "@/lib/api/imports";
import { getErrorMessage } from "@/lib/errors";
import type { ImportStage } from "@/lib/import-data/types";

// ─── Props ─────────────────────────────────────────────────────────────────

interface ImportStageSubmittingProps {
    onStageChange: (stage: ImportStage) => void;
    onClose: () => void;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function ImportStageSubmitting({ onStageChange, onClose }: ImportStageSubmittingProps) {
    const store = useImportStore();
    const jobId = store.meta?.import_job_id ?? null;

    // Flag to prevent double-processing of terminal events
    const terminalHandledRef = React.useRef(false);

    // ─── Handle terminal events ───────────────────────────────────────────
    const handleTerminal = React.useCallback(
        async (state: ImportJobState) => {
            if (terminalHandledRef.current) return;
            terminalHandledRef.current = true;

            try {
                if (state.status === "completed") {
                    // Full success — clear DB and close
                    const schoolId = store.meta?.school_id;
                    if (schoolId) {
                        await store.clearImport(schoolId);
                    }
                    toast.success(
                        `${state.successCount} student${state.successCount !== 1 ? "s" : ""} imported successfully`
                    );
                    onClose();
                    return;
                }

                if (state.status === "completed_with_errors") {
                    // Partial success — fetch failures and reconcile
                    await reconcileFailures(jobId!, store);
                    toast.success(`${state.successCount} students added`, {
                        description: `${state.failedCount} need attention`,
                    });
                    onStageChange("PREVIEW");
                    return;
                }

                if (state.status === "failed") {
                    // Job-level failure — revert to READY
                    await store.setStage("READY");
                    toast.error("Import job failed. You can retry.");
                    onStageChange("READY");
                    return;
                }

                if (state.status === "cancelled") {
                    await store.setStage("READY");
                    toast.error("Import was cancelled.");
                    onStageChange("READY");
                    return;
                }
            } catch (err) {
                toast.error(getErrorMessage(err));
                // Emergency recovery: revert to READY
                await store.setStage("READY");
                onStageChange("READY");
            }
        },
        [jobId, store, onStageChange, onClose]
    );

    // ─── SSE hook ─────────────────────────────────────────────────────────
    const jobState = useImportJobStatus(jobId, handleTerminal);

    // ─── Resume detection ─────────────────────────────────────────────────
    // If we entered SUBMITTING but job_id is null, the POST response was lost
    // but the idempotency_key was persisted. Revert to READY.
    React.useEffect(() => {
        if (!jobId && store.meta?.current_stage === "SUBMITTING") {
            // Check if we have an idempotency key — if so, the POST may have
            // succeeded server-side; let the user retry with the same key
            if (store.meta?.idempotency_key) {
                toast.info("Connection was interrupted. You can retry — your data is safe.");
            }
            store.setStage("READY");
            onStageChange("READY");
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    // ─── Calculate progress percentage ────────────────────────────────────
    const progressPct =
        jobState.totalRecords > 0
            ? Math.round((jobState.processedRecords / jobState.totalRecords) * 100)
            : 0;

    const isProcessing = jobState.status === "pending" || jobState.status === "processing";

    return (
        <div className="flex flex-1 flex-col items-center justify-center gap-6 py-12">
            <div className="flex flex-col items-center gap-2">
                {isProcessing ? (
                    <Loader2 className="text-primary size-8 animate-spin" />
                ) : (
                    <div className="flex items-center">
                        <Badge
                            variant={
                                jobState.status === "completed"
                                    ? "default"
                                    : jobState.status === "completed_with_errors"
                                      ? "secondary"
                                      : "destructive"
                            }
                        >
                            {jobState.status.replace(/_/g, " ")}
                        </Badge>
                    </div>
                )}
                <p className="text-foreground text-lg font-medium">
                    {isProcessing
                        ? "Importing students…"
                        : `Import ${jobState.status.replace(/_/g, " ")}`}
                </p>
            </div>

            {/* Progress bar */}
            <div className="w-full max-w-md space-y-2">
                <Progress value={progressPct} className="h-2" />
                <div className="text-muted-foreground flex justify-between text-xs">
                    <span>
                        {jobState.processedRecords} / {jobState.totalRecords} processed
                    </span>
                    <span>{progressPct}%</span>
                </div>
            </div>

            {/* Stats */}
            <div className="grid grid-cols-3 gap-6 text-center">
                <div>
                    <p className="text-foreground text-2xl font-semibold tracking-tight">
                        {jobState.successCount}
                    </p>
                    <p className="text-muted-foreground text-xs">Imported</p>
                </div>
                <div>
                    <p className="text-destructive text-2xl font-semibold tracking-tight">
                        {jobState.failedCount}
                    </p>
                    <p className="text-muted-foreground text-xs">Failed</p>
                </div>
                <div>
                    <p className="text-muted-foreground text-2xl font-semibold tracking-tight">
                        {jobState.totalRecords}
                    </p>
                    <p className="text-muted-foreground text-xs">Total</p>
                </div>
            </div>

            {/* Progress detail */}
            {isProcessing && (
                <p className="text-muted-foreground text-xs">
                    Processing {jobState.processedChunks} of {jobState.totalChunks} chunks
                </p>
            )}

            {/* Close button for terminal states */}
            {!isProcessing && (
                <Button variant="outline" onClick={onClose}>
                    Close
                </Button>
            )}
        </div>
    );
}

// ─── Failure Reconciliation ────────────────────────────────────────────────

async function reconcileFailures(jobId: string, store: ReturnType<typeof useImportStore>) {
    // Fetch all failures
    const { failures } = await getImportFailures(jobId, { limit: 5000 });

    // Build a set of failed row_numbers from the raw_payload.client_row_ref
    const failedRowNumbers = new Set<number>();

    for (const failure of failures) {
        const clientRowRef = failure.raw_payload?.client_row_ref as string | undefined;
        if (clientRowRef) {
            const rowNum = parseInt(clientRowRef, 10);
            if (!isNaN(rowNum)) {
                failedRowNumbers.add(rowNum);
            }
        }
    }

    // Get all staged rows
    const allRows = await store.getSubmitRows();

    for (const row of allRows) {
        if (failedRowNumbers.has(row.row_number)) {
            // Find the matching failure
            const failure = failures.find((f) => {
                const ref = f.raw_payload?.client_row_ref as string | undefined;
                return ref === String(row.row_number);
            });

            // Mark as error with server rejection details
            await store.updateRow(row.row_number, {
                ui_meta: {
                    ...row.ui_meta,
                    has_error: true,
                    submitted: false,
                    errors: {
                        ...row.ui_meta.errors,
                        server_rejected: failure?.error_message ?? "Server rejected this row",
                        server_error_type: failure?.error_type ?? "SCHEMA_VALIDATION",
                    },
                },
            });
        } else {
            // Mark as submitted — excluded from retry payloads
            await store.updateRow(row.row_number, {
                ui_meta: {
                    ...row.ui_meta,
                    submitted: true,
                },
            });
        }
    }
}
