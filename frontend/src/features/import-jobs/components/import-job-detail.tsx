/**
 * ImportJobDetail — detail view of a single import job.
 *
 * Shows job status, progress stats, cancel action (if active),
 * and failure records (if available).
 */

"use client";

import { CheckCircle2, XCircle, Loader2, Ban, AlertTriangle } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import {
    useImportJobDetail,
    useImportJobFailures,
    useCancelImportJob,
} from "../hooks/use-import-jobs";
import { getErrorMessage } from "@/lib/errors";
import type { ImportJobStatus } from "../types";

// ─── Props ────────────────────────────────────────────────────────────────

interface ImportJobDetailProps {
    jobId: string;
}

// ─── Status helpers ───────────────────────────────────────────────────────

function statusBadge(status: ImportJobStatus) {
    const variants: Record<ImportJobStatus, "default" | "secondary" | "destructive" | "outline"> = {
        pending: "secondary",
        processing: "default",
        completed: "default",
        completed_with_errors: "destructive",
        failed: "destructive",
        cancelling: "secondary",
        cancelled: "outline",
    };
    return <Badge variant={variants[status] ?? "outline"}>{status.replace(/_/g, " ")}</Badge>;
}

const TERMINAL_STATUSES: ImportJobStatus[] = [
    "completed",
    "completed_with_errors",
    "failed",
    "cancelled",
];

// ─── Component ────────────────────────────────────────────────────────────

export function ImportJobDetail({ jobId }: ImportJobDetailProps) {
    const { data: job, isLoading, isError, error } = useImportJobDetail(jobId);
    const { data: failuresData } = useImportJobFailures(jobId, { limit: 200 });
    const cancelMutation = useCancelImportJob();

    // ── Loading ──────────────────────────────────────────────────────────
    if (isLoading) {
        return (
            <div className="space-y-4">
                <Skeleton className="h-8 w-48" />
                <Skeleton className="h-6 w-full" />
                <Skeleton className="h-20 w-full" />
            </div>
        );
    }

    // ── Error ────────────────────────────────────────────────────────────
    if (isError) {
        return (
            <Alert variant="destructive">
                <AlertDescription>{getErrorMessage(error)}</AlertDescription>
            </Alert>
        );
    }

    if (!job) {
        return (
            <Alert>
                <AlertDescription>Import job not found.</AlertDescription>
            </Alert>
        );
    }

    const isTerminal = TERMINAL_STATUSES.includes(job.status);
    const failures = failuresData?.failures ?? [];
    const totalFailed = job.failed_count;
    const allSucceeded = job.status === "completed";
    const isCancelled = job.status === "cancelled";

    const processed = job.success_count + job.failed_count;
    const percentComplete =
        job.total_records > 0
            ? Math.min(100, Math.round((processed / job.total_records) * 100))
            : 0;

    // ── Render ───────────────────────────────────────────────────────────
    return (
        <div className="space-y-6">
            {/* ── Header ──────────────────────────────────────────────── */}
            <div className="space-y-2">
                <div className="flex items-center gap-3">
                    <h1 className="text-foreground text-2xl font-semibold">
                        {job.job_type.replace(/_/g, " ")} Import
                    </h1>
                    {statusBadge(job.status)}
                </div>
                <p className="text-muted-foreground">
                    Created {new Date(job.created_at).toLocaleString()}
                    {job.completed_at &&
                        ` — Completed ${new Date(job.completed_at).toLocaleString()}`}
                </p>
            </div>

            {/* ── Progress ─────────────────────────────────────────────── */}
            <div className="space-y-3">
                <div className="flex items-center justify-between">
                    <span className="text-muted-foreground">
                        {!isTerminal && "Processing…"}
                        {isTerminal && !isCancelled && "Complete"}
                        {isCancelled && "Cancelled"}
                    </span>
                    <span className="font-medium">
                        {processed} / {job.total_records}
                    </span>
                </div>

                <Progress value={percentComplete} className="h-2" />

                <div className="flex flex-wrap items-center gap-4 text-xs">
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
                    {!isTerminal && job.status === "processing" && (
                        <span className="text-muted-foreground flex items-center gap-1">
                            <Loader2 className="size-3.5 animate-spin" />
                            Processing chunk {job.processed_chunks} of {job.total_chunks}
                        </span>
                    )}
                    {job.status === "cancelling" && (
                        <span className="text-muted-foreground flex items-center gap-1">
                            <Ban className="size-3.5" />
                            Waiting for in-flight chunks to finish…
                        </span>
                    )}
                </div>
            </div>

            {/* ── Cancel button ──────────────────────────────────────────── */}
            {job.status === "processing" && (
                <div className="flex justify-end">
                    <Button
                        variant="outline"
                        size="sm"
                        onClick={() => cancelMutation.mutate(jobId)}
                        disabled={cancelMutation.isPending}
                        className="text-destructive hover:text-destructive"
                    >
                        {cancelMutation.isPending ? (
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

            {/* ── Result Banner ──────────────────────────────────────────── */}
            {isTerminal && (
                <div
                    className={`flex items-start gap-3 rounded-md px-4 py-3 ${
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
                        <p className="mt-0.5 text-xs opacity-80">
                            {job.success_count} succeeded, {job.failed_count} failed
                            {job.total_records > 0 && ` out of ${job.total_records} total`}
                        </p>
                    </div>
                </div>
            )}

            {/* ── Failure Details ────────────────────────────────────────── */}
            {failures.length > 0 && (
                <div className="space-y-3">
                    <h2 className="text-foreground text-lg font-medium">
                        Failure Details ({totalFailed} total)
                    </h2>
                    <div className="space-y-2">
                        {failures.slice(0, 100).map((f, i) => (
                            <div key={i} className="bg-muted/30 rounded-md px-3 py-2 text-xs">
                                <div className="flex items-start gap-2">
                                    <XCircle className="text-destructive mt-0.5 size-3 shrink-0" />
                                    <div className="space-y-0.5">
                                        <p>
                                            <span className="text-foreground font-medium">
                                                Row {f.row_number}
                                            </span>
                                            <span className="text-muted-foreground">
                                                {" "}
                                                &mdash; {f.error_type.replace(/_/g, " ")}
                                            </span>
                                        </p>
                                        <p className="text-muted-foreground">{f.error_message}</p>
                                    </div>
                                </div>
                            </div>
                        ))}
                        {failures.length > 100 && (
                            <p className="text-muted-foreground text-xs">
                                …and {failures.length - 100} more failures.
                            </p>
                        )}
                    </div>
                </div>
            )}
        </div>
    );
}
