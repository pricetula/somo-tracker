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

        es.onmessage = (event) => {
            try {
                const parsed = JSON.parse(event.data);
                setProgress(parsed as ProgressData);
                onProgress?.(parsed as ProgressData);
            } catch (e) {
                // Never silently discard parse errors; always log with context.
                console.error("SSE parse error for job", jobId, e);
            }
        };

        es.onerror = () => {
            // No empty catch; reconnect is handled by the browser,
            // but we must not suppress the error silently.
            console.error("SSE connection error for import job", jobId);
        };

        return () => {
            es.close();
        };
    }, [jobId, progressUrl, onProgress]);

    // Optional: expose progress UI when requested (e.g., future resources)
    const progressBar =
        showProgress && progress ? (
            <div className="bg-muted mt-4 rounded border p-3 text-sm">
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
                            onMappedList={mutation.mutate}
                            isSubmitting={mutation.isPending}
                        />
                        {mutation.isError && (
                            <div className="bg-destructive/10 text-destructive mt-4 rounded border p-3 text-sm">
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
                        className="flex h-full w-full items-center justify-center gap-4"
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
