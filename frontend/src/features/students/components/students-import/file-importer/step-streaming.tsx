"use client";

import * as React from "react";
import { CheckCircle2, XCircle, Loader2, AlertCircle, Ban, Pause, Play } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { getErrorMessage } from "@/lib/errors";
import { submitStudentImport, type ImportRequest, type ImportRow } from "@/lib/api/imports";
import {
    getStagedRecordsByStatus,
    updateSessionStep,
    saveSessionMeta,
    assignBatchToRecords,
} from "./db";
import type { StagedStudentRecord, StreamingProgress, ImportSessionMeta } from "./types";

const BATCH_SIZE = 200;
const MAX_CONCURRENT_BATCHES = 3;

interface StepStreamingProps {
    onComplete: () => void;
    onError: (error: string) => void;
}

// ─── Helper: Build ImportRow from StagedStudentRecord ─────────────────────

function toImportRow(record: StagedStudentRecord): ImportRow {
    return {
        full_name: record.payload.full_name,
        gender: (record.payload.gender as "M" | "F") ?? "M",
        date_of_birth: record.payload.date_of_birth ?? null,
        upi_number: record.payload.upi_number ?? null,
        knec_assessment_number: record.payload.knec_assessment_number ?? null,
    };
}

// ─── Batch Worker ─────────────────────────────────────────────────────────

async function processBatch(
    batch: StagedStudentRecord[],
    batchIndex: number,
    signal: AbortSignal,
    onBatchComplete: (success: number, failed: number, batchId: string) => void,
    onError: (err: string) => void
): Promise<void> {
    if (signal.aborted) return;

    const batchId = `batch_${batchIndex}_${Date.now()}`;
    const rows = batch.map(toImportRow);

    try {
        const payload: ImportRequest = {
            rows,
            academic_term_id: "", // backend will determine or require this
        };

        const response = await submitStudentImport(payload);

        // Mark records as submitted
        const recordIds = batch.map((r) => r.id).filter((id): id is number => id !== undefined);
        await assignBatchToRecords(recordIds, batchId);

        onBatchComplete(
            response.status === "completed" ? rows.length : 0,
            response.status === "completed_with_errors" ? rows.length : 0,
            batchId
        );
    } catch (err) {
        if (signal.aborted) return;
        onError(`Batch ${batchIndex + 1} failed: ${getErrorMessage(err)}`);
        // Still mark as partial failure
        onBatchComplete(0, rows.length, batchId);
    }
}

// ─── Main Component ───────────────────────────────────────────────────────

export function StepStreaming({ onComplete, onError }: StepStreamingProps) {
    const [progress, setProgress] = React.useState<StreamingProgress>({
        total_batches: 0,
        completed_batches: 0,
        success_count: 0,
        failed_count: 0,
        current_batch: 0,
        status: "idle",
    });
    const [records, setRecords] = React.useState<StagedStudentRecord[]>([]);
    const [errorMessage, setErrorMessage] = React.useState<string | null>(null);
    const [paused, setPaused] = React.useState(false);
    const abortRef = React.useRef<AbortController | null>(null);

    // Load valid records on mount
    React.useEffect(() => {
        getStagedRecordsByStatus("valid").then((valid) => {
            const batches = Math.ceil(valid.length / BATCH_SIZE);
            setRecords(valid);
            setProgress((prev) => ({
                ...prev,
                total_batches: batches,
            }));
        });
    }, []);

    const startStreaming = React.useCallback(async () => {
        if (records.length === 0) {
            onError("No valid records to import.");
            return;
        }

        const abortController = new AbortController();
        abortRef.current = abortController;

        await updateSessionStep("streaming");
        setPaused(false);
        setProgress((prev) => ({ ...prev, status: "streaming" }));

        // Split into batches
        const batches: StagedStudentRecord[][] = [];
        for (let i = 0; i < records.length; i += BATCH_SIZE) {
            batches.push(records.slice(i, i + BATCH_SIZE));
        }

        let completed = 0;
        let successCount = 0;
        let failedCount = 0;
        const completedBatchIds: string[] = [];

        // Process batches with limited concurrency
        for (let i = 0; i < batches.length; i += MAX_CONCURRENT_BATCHES) {
            if (abortController.signal.aborted) break;

            // Wait if paused
            while (paused) {
                await new Promise((resolve) => setTimeout(resolve, 200));
                if (abortController.signal.aborted) break;
            }

            const batchGroup = batches.slice(i, i + MAX_CONCURRENT_BATCHES);

            await Promise.all(
                batchGroup.map((batch, idx) =>
                    processBatch(
                        batch,
                        i + idx,
                        abortController.signal,
                        (s, f, batchId) => {
                            completed++;
                            successCount += s;
                            failedCount += f;
                            completedBatchIds.push(batchId);

                            setProgress((prev) => ({
                                ...prev,
                                completed_batches: completed,
                                success_count: successCount,
                                failed_count: failedCount,
                                current_batch: i + idx + 1,
                            }));

                            // Persist completed_batch_ids
                            const batchMeta: Partial<ImportSessionMeta> = {
                                completed_batch_ids: completedBatchIds,
                            };
                            saveSessionMeta(batchMeta).catch(() => {});
                        },
                        (err) => {
                            setErrorMessage((prev) => (prev ? `${prev}\n${err}` : err));
                        }
                    )
                )
            );
        }

        if (!abortController.signal.aborted) {
            setProgress((prev) => ({
                ...prev,
                status: "completed" as const,
            }));

            // Clear staging after completion
            const { clearAllSessions } = await import("./db");
            await clearAllSessions();

            if (failedCount > 0) {
                onError(
                    `Import completed with ${failedCount} failure(s). ${successCount} students imported successfully.`
                );
            }

            setTimeout(() => onComplete(), 1500);
        }
    }, [records, paused, onComplete, onError]);

    const handlePause = React.useCallback(() => {
        setPaused(true);
        setProgress((prev) => ({ ...prev, status: "paused" }));
    }, []);

    const handleResume = React.useCallback(() => {
        setPaused(false);
        setProgress((prev) => ({ ...prev, status: "streaming" }));
    }, []);

    const handleCancel = React.useCallback(() => {
        if (abortRef.current) {
            abortRef.current.abort();
        }
        setProgress((prev) => ({ ...prev, status: "failed" }));
    }, []);

    // Cleanup on unmount
    React.useEffect(() => {
        return () => {
            abortRef.current?.abort();
        };
    }, []);

    const percentComplete = React.useMemo(() => {
        if (progress.total_batches === 0) return 0;
        return Math.round((progress.completed_batches / progress.total_batches) * 100);
    }, [progress.completed_batches, progress.total_batches]);

    // ─── Render ────────────────────────────────────────────────────────

    return (
        <div className="space-y-6">
            <div>
                <h3 className="text-sm font-medium">Importing Records</h3>
                <p className="text-muted-foreground mt-1 text-xs">
                    Streaming {records.length} student records to the server in batches of{" "}
                    {BATCH_SIZE}.
                </p>
            </div>

            {/* Progress */}
            <div className="space-y-3">
                <div className="flex items-center justify-between text-sm">
                    <span className="text-muted-foreground">
                        Batch {progress.current_batch} of {progress.total_batches}
                    </span>
                    <span className="font-medium">{percentComplete}%</span>
                </div>
                <Progress value={percentComplete} className="h-2" />

                <div className="flex items-center gap-4 text-xs">
                    <span className="flex items-center gap-1 text-emerald-600">
                        <CheckCircle2 className="size-3.5" />
                        {progress.success_count} succeeded
                    </span>
                    {progress.failed_count > 0 && (
                        <span className="text-destructive flex items-center gap-1">
                            <XCircle className="size-3.5" />
                            {progress.failed_count} failed
                        </span>
                    )}
                    {progress.status === "streaming" && (
                        <span className="text-muted-foreground flex items-center gap-1">
                            <Loader2 className="size-3.5 animate-spin" />
                            Streaming...
                        </span>
                    )}
                </div>
            </div>

            {/* Error alert */}
            {errorMessage && (
                <Alert variant="destructive">
                    <AlertCircle className="size-4" />
                    <AlertTitle>Errors during import</AlertTitle>
                    <AlertDescription className="whitespace-pre-wrap">
                        {errorMessage}
                    </AlertDescription>
                </Alert>
            )}

            {/* Actions */}
            <div className="flex items-center justify-between">
                {progress.status === "idle" && (
                    <Button size="sm" onClick={startStreaming}>
                        Start Import
                    </Button>
                )}

                {progress.status === "streaming" && (
                    <div className="flex items-center gap-2">
                        <Button variant="outline" size="sm" onClick={handlePause}>
                            <Pause className="mr-1.5 size-3.5" />
                            Pause
                        </Button>
                        <Button variant="destructive" size="sm" onClick={handleCancel}>
                            <Ban className="mr-1.5 size-3.5" />
                            Cancel
                        </Button>
                    </div>
                )}

                {progress.status === "paused" && (
                    <Button size="sm" onClick={handleResume}>
                        <Play className="mr-1.5 size-3.5" />
                        Resume
                    </Button>
                )}

                {progress.status === "completed" && (
                    <Button size="sm" onClick={onComplete}>
                        Done
                    </Button>
                )}

                {progress.status === "failed" && (
                    <Button size="sm" onClick={startStreaming}>
                        Retry
                    </Button>
                )}
            </div>
        </div>
    );
}
