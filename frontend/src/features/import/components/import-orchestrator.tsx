"use client";

import React from "react";
import { motion, AnimatePresence } from "framer-motion";
import { useMutation } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { Upload } from "./upload";
import { ManualImport } from "./manual-import";
import { FieldDef, MappedRow } from "./upload/field-mapper";

export interface ProgressData {
    status: string;
    succeeded: number;
    failed: number;
    deferred?: number;
    total: number;
}

interface ImportOrchestratorProps {
    fieldDef: FieldDef[];
    onMappedList: (
        rows: MappedRow[]
    ) => Promise<{ job_id: string; total_records?: number; status?: string }>;
    progressUrl?: (jobId: string) => string;
    showProgress?: boolean;
    onProgress?: (data: ProgressData) => void;
    onReset?: () => void;
    isSubmitting?: boolean;
    [key: string]: unknown;
}

/**
 * ImportOrchestrator — flexible central state machine for data import flows.
 *
 * Designed to be reused across resources (parents, exams, classes) without
 * backend rewrites. The parent provides onSubmit / progressUrl props;
 * the component owns mutation + SSE progress tracking.
 */
export function ImportOrchestrator({
    fieldDef,
    onMappedList,
    progressUrl,
    showProgress,
    onProgress,
    onReset,
}: ImportOrchestratorProps) {
    const [importType, setImportType] = React.useState("");
    const [jobId, setJobId] = React.useState<string | null>(null);
    const [progress, setProgress] = React.useState<ProgressData | null>(null);

    const mutation = useMutation({
        mutationFn: async (rows: MappedRow[]) => await onMappedList(rows),
        onSuccess: (data) => {
            if (data?.job_id) {
                setJobId(data.job_id);
            }
        },
        onError: (err: Error) => {
            // Log once at handler layer; never both log and return, never silent.
            console.error("ImportOrchestrator mutation error:", err.message || err);
        },
    });

    // SSE progress stream — opens only when jobId is present
    React.useEffect(() => {
        if (!jobId || !progressUrl) return;
        const url = progressUrl(jobId);
        const es = new EventSource(url);

        const handleProgress = (event: MessageEvent) => {
            try {
                const parsed = JSON.parse(event.data);
                setProgress(parsed as ProgressData);
                onProgress?.(parsed as ProgressData);
            } catch (e) {
                console.error("SSE parse error for job", jobId, e);
            }
        };

        es.addEventListener("progress", handleProgress);
        es.onmessage = handleProgress; // fallback for events without type

        es.onerror = () => {
            console.error("SSE connection error for import job", jobId);
        };

        return () => {
            es.removeEventListener("progress", handleProgress);
            es.close();
        };
    }, [jobId, progressUrl, onProgress]);

    // Progress view when job is running
    if (jobId && showProgress) {
        const isDone =
            progress &&
            (progress.status === "COMPLETED" || progress.status === "COMPLETED_WITH_ERRORS");
        const progressContent = progress ? (
            <div className="bg-muted rounded border p-4">
                <div className="mb-2 font-medium">Job {jobId}</div>
                <div className="">Status: {progress.status}</div>
                <div className="mt-1">
                    {progress.succeeded} / {progress.total} succeeded
                    {progress.failed > 0 && `, ${progress.failed} failed`}
                    {progress.deferred ? `, ${progress.deferred} deferred` : ""}
                </div>
                <div className="bg-muted-foreground/20 mt-3 h-2 w-full overflow-hidden rounded">
                    <div
                        className="h-full bg-emerald-500 transition-all"
                        style={{
                            width: `${progress.total ? Math.round((progress.succeeded / progress.total) * 100) : 0}%`,
                        }}
                    />
                </div>
                {isDone && (
                    <div className="mt-4 flex gap-2">
                        <Button
                            size="sm"
                            variant="outline"
                            onClick={() => {
                                setJobId(null);
                                setProgress(null);
                                setImportType("");
                                onReset?.();
                            }}
                        >
                            New import
                        </Button>
                        <Button
                            size="sm"
                            variant="secondary"
                            onClick={() => {
                                setProgress(null);
                            }}
                        >
                            Hide progress
                        </Button>
                    </div>
                )}
            </div>
        ) : (
            <div className="bg-muted rounded border p-4">Starting job…</div>
        );
        return (
            <div className="relative flex max-w-4xl gap-4 overflow-hidden">
                <div className="h-full w-full">{progressContent}</div>
            </div>
        );
    }

    // Optional: expose progress UI when requested (e.g., future resources)
    const progressBar =
        showProgress && progress ? (
            <div className="bg-muted mt-4 rounded border p-3">
                <div className="font-medium">Progress: {progress.status}</div>
                <div>
                    {progress.succeeded} / {progress.total} succeeded
                    {progress.failed > 0 && `, ${progress.failed} failed`}
                </div>
            </div>
        ) : null;

    return (
        <div className="relative flex max-w-4xl gap-4 overflow-hidden">
            <AnimatePresence mode="wait">
                {importType === "manual" ? (
                    <motion.div
                        key="manual"
                        initial={{ opacity: 0, y: 12, scale: 0.98 }}
                        animate={{ opacity: 1, y: 0, scale: 1 }}
                        exit={{ opacity: 0, y: -12, scale: 0.98 }}
                        transition={{ duration: 0.25, ease: "easeInOut" }}
                        className="h-full w-full"
                    >
                        <ManualImport onCancel={() => setImportType("")} />
                    </motion.div>
                ) : importType === "upload" ? (
                    <motion.div
                        key="upload"
                        initial={{ opacity: 0, y: 12, scale: 0.98 }}
                        animate={{ opacity: 1, y: 0, scale: 1 }}
                        exit={{ opacity: 0, y: -12, scale: 0.98 }}
                        transition={{ duration: 0.25, ease: "easeInOut" }}
                        className="h-full w-full"
                    >
                        <Upload
                            onCancel={() => setImportType("")}
                            fieldDef={fieldDef}
                            onMappedList={mutation.mutateAsync}
                            isSubmitting={mutation.isPending}
                        />
                        {mutation.isError && (
                            <div className="bg-destructive/10 text-destructive mt-4 rounded border p-3">
                                Submit failed. Please retry.
                            </div>
                        )}
                    </motion.div>
                ) : (
                    <motion.div
                        key="select"
                        initial={{ opacity: 0, y: 12, scale: 0.98 }}
                        animate={{ opacity: 1, y: 0, scale: 1 }}
                        exit={{ opacity: 0, y: -12, scale: 0.98 }}
                        transition={{ duration: 0.25, ease: "easeInOut" }}
                        className="flex h-58 w-full items-center justify-center gap-4"
                    >
                        <Button onClick={() => setImportType("upload")}>Upload</Button>
                        <Button variant="outline" onClick={() => setImportType("manual")}>
                            Manual Import
                        </Button>
                    </motion.div>
                )}
            </AnimatePresence>
            {progressBar}
        </div>
    );
}
