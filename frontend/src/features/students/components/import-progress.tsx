/**
 * Import Progress Widget — miniature bottom-right status widget
 * (Google Drive upload style) that tracks SSE progress for CSV imports.
 */

"use client";

import * as React from "react";
import { Progress } from "@/components/ui/progress";
import { Button } from "@/components/ui/button";
import { X, CheckCircle2, AlertCircle, Loader2, Download } from "lucide-react";
import { cn } from "@/lib/utils";

import { downloadErrorCSV } from "@/lib/api/students";

// ─── Types ─────────────────────────────────────────────────────────────────

type ImportStatus = "uploading" | "processing" | "completed" | "error";

interface ImportState {
    importId: string;
    status: ImportStatus;
    current: number;
    total: number;
    success: number;
    failed: number;
    downloadUrl?: string;
    errorMessage?: string;
    connectionLost?: boolean;
}

// ─── Props ─────────────────────────────────────────────────────────────────

interface ImportProgressProps {
    importId: string;
    onDismiss: () => void;
    onComplete: () => void;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function ImportProgress({ importId, onDismiss, onComplete }: ImportProgressProps) {
    const [state, setState] = React.useState<ImportState>({
        importId,
        status: "processing",
        current: 0,
        total: 0,
        success: 0,
        failed: 0,
    });
    const [show, setShow] = React.useState(true);
    const [isDownloading, setIsDownloading] = React.useState(false);

    React.useEffect(() => {
        const apiBase = process.env.NEXT_PUBLIC_API_URL ?? "";
        const eventSource = new EventSource(
            `${apiBase}/api/v1/students/import/stream?id=${encodeURIComponent(importId)}`,
            { withCredentials: true }
        );

        eventSource.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data);
                setState((prev) => ({
                    ...prev,
                    status: data.status,
                    current: data.current ?? prev.current,
                    total: data.total ?? prev.total,
                    success: data.success ?? prev.success,
                    failed: data.failed ?? prev.failed,
                    downloadUrl: data.download_url ?? prev.downloadUrl,
                    errorMessage: data.error ?? prev.errorMessage,
                    connectionLost: false,
                }));

                if (data.status === "completed" || data.status === "error") {
                    eventSource.close();
                    if (data.status === "completed") {
                        onComplete();
                    }
                }
            } catch {
                // Ignore parse errors
            }
        };

        eventSource.onerror = () => {
            // EventSource auto-reconnects, but we surface a warning
            setState((prev) => ({
                ...prev,
                connectionLost: true,
            }));
        };

        return () => {
            eventSource.close();
        };
    }, [importId, onComplete]);

    // Compute progress percentage
    const percentage =
        state.total > 0
            ? Math.round(((state.current > 0 ? state.current : state.total) / state.total) * 100)
            : 0;

    const isTerminal = state.status === "completed" || state.status === "error";

    if (!show) return null;

    async function handleDownload() {
        if (!state.downloadUrl) return;
        setIsDownloading(true);
        try {
            // Extract error ID from download URL
            const url = new URL(state.downloadUrl, "http://localhost");
            const errorId = url.searchParams.get("id");
            if (!errorId) return;

            const csvContent = await downloadErrorCSV(errorId);
            const blob = new Blob([csvContent], { type: "text/csv" });
            const blobUrl = URL.createObjectURL(blob);
            const a = document.createElement("a");
            a.href = blobUrl;
            a.download = `import_errors_${errorId}.csv`;
            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);
            URL.revokeObjectURL(blobUrl);
        } catch {
            // Silently fail
        } finally {
            setIsDownloading(false);
        }
    }

    return (
        <div
            className={cn(
                "bg-background fixed right-4 bottom-4 z-50 w-80 rounded-lg border p-4 shadow-lg transition-all duration-300",
                isTerminal && "border-green-200 dark:border-green-800"
            )}
        >
            {/* Header */}
            <div className="mb-2 flex items-center justify-between">
                <div className="flex items-center gap-2">
                    {state.status === "processing" && !state.connectionLost && (
                        <Loader2 className="text-muted-foreground size-4 animate-spin" />
                    )}
                    {state.connectionLost && <AlertCircle className="size-4 text-amber-500" />}
                    {state.status === "completed" && (
                        <CheckCircle2 className="size-4 text-green-500" />
                    )}
                    {state.status === "error" && (
                        <AlertCircle className="text-destructive size-4" />
                    )}
                    <span className="text-xs font-medium">
                        {state.connectionLost
                            ? "Reconnecting..."
                            : state.status === "completed"
                              ? "Import complete"
                              : state.status === "error"
                                ? "Import failed"
                                : "Importing students..."}
                    </span>
                </div>
                <button
                    onClick={() => {
                        setShow(false);
                        onDismiss();
                    }}
                    className="text-muted-foreground hover:text-foreground transition-colors"
                >
                    <X className="size-3.5" />
                </button>
            </div>

            {/* Progress bar */}
            <Progress
                value={percentage}
                className={cn(
                    "mb-2 h-1.5",
                    state.status === "completed" && "bg-green-100 [&>div]:bg-green-500",
                    state.connectionLost && "bg-amber-100 [&>div]:bg-amber-500"
                )}
            />

            {/* Stats */}
            <div className="text-muted-foreground flex items-center justify-between text-xs">
                <span>
                    {state.current > 0
                        ? `${state.current} / ${state.total} rows`
                        : `${state.total || "..."} rows`}
                </span>
                <span>{percentage}%</span>
            </div>

            {/* Error download link */}
            {state.status === "completed" && state.failed > 0 && state.downloadUrl && (
                <div className="border-border/40 mt-2 border-t pt-2">
                    <Button
                        variant="ghost"
                        size="sm"
                        className="text-destructive hover:text-destructive h-7 w-full justify-start text-xs"
                        onClick={handleDownload}
                        disabled={isDownloading}
                    >
                        <Download className="mr-1.5 size-3" />
                        {isDownloading
                            ? "Downloading..."
                            : `Download errors (${state.failed} rows)`}
                    </Button>
                </div>
            )}

            {/* Reconnection notice */}
            {state.connectionLost && (
                <p className="mt-1 text-xs text-amber-600">
                    Connection interrupted — reconnecting automatically...
                </p>
            )}
        </div>
    );
}
