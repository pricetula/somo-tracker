"use client";

/**
 * ImportProgress — polls a running import job and renders its progress.
 *
 * Shared by the bulk invite flow (manual + file import). After a job is
 * created the parent hands over to this component, which polls
 * GET /api/v1/imports/:job_id until the job reaches a terminal status and
 * then calls onDone (or onRetry for failures).
 */

import * as React from "react";
import { AlertCircle, CheckCircle2, Loader2, XCircle } from "lucide-react";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { getImportJob, type ImportJob, type ImportJobStatus } from "@/lib/api/imports";

const POLL_INTERVAL_MS = 2500;
const MAX_POLL_ERRORS = 5;
const AUTO_CLOSE_DELAY_MS = 1500;

const TERMINAL_STATUSES: ReadonlySet<ImportJobStatus> = new Set([
    "completed",
    "completed_with_errors",
    "failed",
    "cancelled",
]);

interface ImportProgressProps {
    jobId: string;
    totalRecords: number;
    onDone: () => void;
    onRetry: () => void;
}

export function ImportProgress({ jobId, totalRecords, onDone, onRetry }: ImportProgressProps) {
    const [job, setJob] = React.useState<ImportJob | null>(null);
    const [pollFailed, setPollFailed] = React.useState(false);
    const [refreshKey, setRefreshKey] = React.useState(0);
    const autoClosedRef = React.useRef(false);
    const onDoneRef = React.useRef(onDone);
    // Keep the latest onDone in a ref without touching it during render.
    React.useEffect(() => {
        onDoneRef.current = onDone;
    });

    React.useEffect(() => {
        let cancelled = false;
        let pollErrors = 0;
        let timer: ReturnType<typeof setTimeout>;

        const poll = async () => {
            if (cancelled) return;
            try {
                const next = await getImportJob(jobId);
                if (cancelled) return;
                setJob(next);
                pollErrors = 0;
                setPollFailed(false);

                if (TERMINAL_STATUSES.has(next.status)) {
                    // Auto-close successful/neutral terminal states after a beat
                    // so the user sees the final status; failures wait for input.
                    if (next.status !== "failed" && !autoClosedRef.current) {
                        autoClosedRef.current = true;
                        timer = setTimeout(() => onDoneRef.current(), AUTO_CLOSE_DELAY_MS);
                    }
                    return;
                }
            } catch (err) {
                if (cancelled) return;
                pollErrors += 1;
                if (pollErrors >= MAX_POLL_ERRORS) {
                    setPollFailed(true);
                    return;
                }
                console.warn("Import progress poll failed; will retry.", err);
            }
            timer = setTimeout(poll, POLL_INTERVAL_MS);
        };

        timer = setTimeout(poll, 0);

        return () => {
            cancelled = true;
            clearTimeout(timer);
        };
    }, [jobId, refreshKey]);

    // ── Derived values ──────────────────────────────────────────────────

    const total = job?.total_records ?? totalRecords;
    const processed = job?.processed_records ?? 0;
    const pct = total > 0 ? Math.min(100, Math.round((processed / total) * 100)) : 0;
    const status = job?.status;

    // ── Render ──────────────────────────────────────────────────────────

    if (pollFailed) {
        return (
            <Alert>
                <AlertCircle className="size-4" />
                <AlertTitle>Status updates paused</AlertTitle>
                <AlertDescription>
                    We lost the connection to the import job. You can resume checking below.
                </AlertDescription>
                <div className="mt-3 flex items-center gap-2">
                    <Button
                        size="sm"
                        onClick={() => {
                            setPollFailed(false);
                            setRefreshKey((k) => k + 1);
                        }}
                    >
                        Resume Checking
                    </Button>
                </div>
            </Alert>
        );
    }

    if (!job) {
        return (
            <div className="flex items-center justify-center gap-2 py-8">
                <Loader2 className="text-muted-foreground size-5 animate-spin" />
                <p className="text-muted-foreground text-xs">Checking import status...</p>
            </div>
        );
    }

    if (status === "failed") {
        return (
            <Alert variant="destructive">
                <XCircle className="size-4" />
                <AlertTitle>Import failed</AlertTitle>
                <AlertDescription>
                    The import job did not complete successfully. Please try again.
                </AlertDescription>
                <div className="mt-3 flex items-center gap-2">
                    <Button size="sm" onClick={onRetry}>
                        Try Again
                    </Button>
                    <Button size="sm" variant="ghost" onClick={onDone}>
                        Close
                    </Button>
                </div>
            </Alert>
        );
    }

    if (status === "completed") {
        return (
            <Alert>
                <CheckCircle2 className="size-4" />
                <AlertTitle>Import completed</AlertTitle>
                <AlertDescription>
                    Successfully imported {job.success_count} of {total} record
                    {total !== 1 ? "s" : ""}.
                </AlertDescription>
                <div className="mt-3">
                    <Button size="sm" onClick={onDone}>
                        Done
                    </Button>
                </div>
            </Alert>
        );
    }

    if (status === "completed_with_errors") {
        return (
            <Alert>
                <AlertCircle className="size-4" />
                <AlertTitle>Import completed with errors</AlertTitle>
                <AlertDescription>
                    {job.success_count} record{job.success_count !== 1 ? "s" : ""} imported,{" "}
                    {job.failed_count} failed.
                </AlertDescription>
                <div className="mt-3">
                    <Button size="sm" onClick={onDone}>
                        Done
                    </Button>
                </div>
            </Alert>
        );
    }

    if (status === "cancelled") {
        return (
            <Alert>
                <AlertCircle className="size-4" />
                <AlertTitle>Import cancelled</AlertTitle>
                <AlertDescription>This import was cancelled.</AlertDescription>
                <div className="mt-3">
                    <Button size="sm" variant="ghost" onClick={onDone}>
                        Close
                    </Button>
                </div>
            </Alert>
        );
    }

    if (status === "cancelling") {
        return (
            <div className="space-y-3">
                <div className="flex items-center gap-2">
                    <Loader2 className="text-muted-foreground size-4 animate-spin" />
                    <p className="font-medium">Cancelling import…</p>
                </div>
                <p className="text-muted-foreground text-xs">This may take a moment.</p>
            </div>
        );
    }

    // pending / processing
    return (
        <div className="space-y-3">
            <div className="flex items-center gap-2">
                <Loader2 className="text-muted-foreground size-4 animate-spin" />
                <p className="font-medium">Import in progress</p>
            </div>
            <Progress value={pct} className="h-2" />
            <p className="text-muted-foreground text-xs">
                {processed.toLocaleString()} of {total.toLocaleString()} record
                {total !== 1 ? "s" : ""} processed · {job.success_count} succeeded ·{" "}
                {job.failed_count} failed
            </p>
        </div>
    );
}
