/**
 * Post-submit results summary screen.
 *
 * Displays success confirmation or partial-failure details with retry actions.
 */

"use client";

import { AlertCircle, CheckCircle, RefreshCw } from "lucide-react";
import type { ImportResultSummary } from "../types";

interface ResultsSummaryProps {
    summary: ImportResultSummary;
    onRetry: () => void;
    onStartNew: () => void;
}

export function ResultsSummary({ summary, onRetry, onStartNew }: ResultsSummaryProps) {
    if (summary.status === "success") {
        return (
            <div className="space-y-4 py-8 text-center">
                <CheckCircle className="mx-auto size-12 text-emerald-500" />
                <h2 className="text-lg font-semibold">Import Complete</h2>
                <p className="text-muted-foreground text-sm">
                    Successfully imported{" "}
                    <span className="text-foreground font-medium">{summary.successCount}</span>{" "}
                    student{summary.successCount !== 1 ? "s" : ""}.
                </p>
                <button
                    onClick={onStartNew}
                    className="bg-primary text-primary-foreground hover:bg-primary/90 mx-auto mt-4 rounded-md px-5 py-1.5 text-sm font-medium"
                >
                    Start New Import
                </button>
            </div>
        );
    }

    if (summary.status === "partial") {
        return (
            <div className="space-y-4">
                <div className="flex items-center gap-3">
                    <AlertCircle className="size-6 text-emerald-500" />
                    <div>
                        <h2 className="text-lg font-semibold">Partial Success</h2>
                        <p className="text-muted-foreground text-sm">
                            {summary.successCount} student{summary.successCount !== 1 ? "s" : ""}{" "}
                            imported. {summary.failureCount} failed.
                        </p>
                    </div>
                </div>

                {summary.failures.length > 0 && (
                    <div className="space-y-2">
                        <p className="text-muted-foreground text-xs font-medium">
                            Failed records — review and retry:
                        </p>
                        <div className="max-h-80 space-y-1 overflow-auto">
                            {summary.failures.map((failure, idx) => (
                                <div
                                    key={idx}
                                    className="bg-muted/20 flex items-start gap-2 rounded-md px-3 py-2"
                                >
                                    <AlertCircle className="text-destructive mt-0.5 size-4 shrink-0" />
                                    <div className="text-sm">
                                        <p className="font-medium">{failure.full_name}</p>
                                        <p className="text-destructive text-xs">
                                            {failure.error_message ?? "Unknown error"}
                                        </p>
                                        {failure.field_errors &&
                                            Object.entries(failure.field_errors).map(
                                                ([field, msg]) => (
                                                    <p
                                                        key={field}
                                                        className="text-muted-foreground text-xs"
                                                    >
                                                        {field}: {msg}
                                                    </p>
                                                )
                                            )}
                                    </div>
                                </div>
                            ))}
                        </div>
                    </div>
                )}

                <div className="flex items-center justify-between pt-2">
                    <button
                        onClick={onStartNew}
                        className="text-muted-foreground hover:text-foreground text-sm"
                    >
                        Start New Import
                    </button>
                    <button
                        onClick={onRetry}
                        className="bg-primary text-primary-foreground hover:bg-primary/90 flex items-center gap-1.5 rounded-md px-4 py-1.5 text-sm font-medium"
                    >
                        <RefreshCw className="size-3.5" />
                        Retry Failed
                    </button>
                </div>
            </div>
        );
    }

    // Error state
    return (
        <div className="space-y-4 py-8 text-center">
            <AlertCircle className="text-destructive mx-auto size-12" />
            <h2 className="text-lg font-semibold">Import Failed</h2>
            <p className="text-muted-foreground text-sm">
                {summary.message ?? "An unexpected error occurred during submission."}
            </p>
            <div className="flex items-center justify-center gap-3 pt-4">
                <button
                    onClick={onRetry}
                    className="bg-primary text-primary-foreground hover:bg-primary/90 flex items-center gap-1.5 rounded-md px-4 py-1.5 text-sm font-medium"
                >
                    <RefreshCw className="size-3.5" />
                    Retry
                </button>
                <button
                    onClick={onStartNew}
                    className="text-muted-foreground hover:text-foreground text-sm"
                >
                    Start New Import
                </button>
            </div>
        </div>
    );
}
