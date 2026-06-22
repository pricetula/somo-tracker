/**
 * Import Progress Panel — real-time SSE progress tracking with RxJS.
 *
 * Subscribes to the SSE endpoint and shows a live progress bar + counts.
 * Falls back to polling if SSE connection drops.
 * On completion, shows a summary with option to correct failed rows.
 */

"use client";

import * as React from "react";
import { Loader2, CheckCircle2, AlertTriangle } from "lucide-react";

import { createImportProgressStream } from "../hooks/use-staff-import";
import type { ImportProgressEvent } from "@/lib/api/imports";

// ─── Types ─────────────────────────────────────────────────────────────────

interface ImportProgressPanelProps {
    jobID: string;
    onDone: (hasErrors: boolean) => void;
    onClose: () => void;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function ImportProgressPanel({ jobID, onDone, onClose }: ImportProgressPanelProps) {
    const [status, setStatus] = React.useState<string>("processing");
    const [processed, setProcessed] = React.useState(0);
    const [success, setSuccess] = React.useState(0);
    const [failed, setFailed] = React.useState(0);
    const [total, setTotal] = React.useState(0);
    const [finished, setFinished] = React.useState(false);

    React.useEffect(() => {
        const subscription = createImportProgressStream(jobID).subscribe({
            next: (event: ImportProgressEvent) => {
                setStatus(event.status ?? "processing");
                setProcessed(event.processed_records ?? 0);
                setSuccess(event.success_count ?? 0);
                setFailed(event.failed_count ?? 0);
                setTotal(event.total_records ?? 0);

                if (event.type === "import_finished") {
                    setFinished(true);
                }
            },
            error: () => {
                // Subscription will re-subscribe via fallback polling
            },
        });

        return () => subscription.unsubscribe();
    }, [jobID]);

    // Notify parent when finished
    React.useEffect(() => {
        if (finished) {
            const timer = setTimeout(() => {
                onDone(failed > 0);
            }, 1000);
            return () => clearTimeout(timer);
        }
    }, [finished, failed, onDone]);

    const progress = total > 0 ? Math.round((processed / total) * 100) : 0;
    const isProcessing = !finished && (status === "pending" || status === "processing");

    return (
        <div className="flex flex-col items-center justify-center gap-6 py-12">
            {isProcessing ? (
                <>
                    <Loader2 className="text-primary size-10 animate-spin" />
                    <p className="text-lg font-medium">Processing your import...</p>

                    {/* Progress bar */}
                    <div className="bg-muted h-2 w-full max-w-md overflow-hidden rounded-full">
                        <div
                            className="bg-primary h-full rounded-full transition-all duration-500"
                            style={{ width: `${Math.max(progress, 5)}%` }}
                        />
                    </div>

                    <div className="text-muted-foreground flex gap-6 text-sm">
                        <span>
                            {processed} / {total} records
                        </span>
                        {success > 0 && <span className="text-emerald-600">{success} sent</span>}
                        {failed > 0 && <span className="text-destructive">{failed} failed</span>}
                    </div>
                </>
            ) : (
                <>
                    {failed > 0 ? (
                        <AlertTriangle className="text-destructive size-10" />
                    ) : (
                        <CheckCircle2 className="size-10 text-emerald-600" />
                    )}

                    <p className="text-lg font-medium">
                        {failed > 0 ? "Completed with errors" : "Import complete!"}
                    </p>

                    <div className="text-muted-foreground flex gap-6 text-sm">
                        <span>Total: {total}</span>
                        <span className="text-emerald-600">Sent: {success}</span>
                        {failed > 0 && <span className="text-destructive">Failed: {failed}</span>}
                    </div>

                    <div className="flex gap-3">
                        {failed > 0 && (
                            <button
                                onClick={() => onDone(true)}
                                className="bg-primary text-primary-foreground hover:bg-primary/90 rounded-md px-4 py-2 text-sm font-medium"
                            >
                                Review Failed
                            </button>
                        )}
                        <button
                            onClick={onClose}
                            className="text-muted-foreground hover:text-foreground rounded-md px-4 py-2 text-sm font-medium"
                        >
                            Close
                        </button>
                    </div>
                </>
            )}
        </div>
    );
}
