/**
 * Stage 3: Ready to Dispatch.
 *
 * - Final summary card
 * - "Upload" button gated on zero unresolved errors
 * - Generates idempotency_key before POST
 * - Transitions to SUBMITTING immediately after persisting key
 */

"use client";

import * as React from "react";

import { Button } from "@/components/ui/button";
import { Loader2 } from "lucide-react";

import { useImportStore } from "../hooks/use-import-store";
import { submitStudentImport, type ImportResponse } from "@/lib/api/imports";
import { getErrorMessage } from "@/lib/errors";
import type { ImportStage } from "@/lib/import-data/types";

// ─── Props ─────────────────────────────────────────────────────────────────

interface ImportStageReadyProps {
    onStageChange: (stage: ImportStage, jobId?: string) => void;
    onClose: () => void;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function ImportStageReady({ onStageChange, onClose }: ImportStageReadyProps) {
    const store = useImportStore();
    const [submitting, setSubmitting] = React.useState(false);
    const [error, setError] = React.useState<string | null>(null);

    const totalRows = store.meta?.total_rows ?? 0;
    const toSubmit = totalRows - store.skippedCount;
    const hasErrors = store.errorCount > 0;

    // ─── Upload handler ───────────────────────────────────────────────────
    const handleUpload = async () => {
        if (hasErrors || submitting) return;
        setSubmitting(true);
        setError(null);

        try {
            // Step 1: Generate and persist idempotency_key BEFORE the POST
            const idempotencyKey = crypto.randomUUID();
            await store.setIdempotencyKey(idempotencyKey);

            // Step 2: Persist SUBMITTING stage before the POST
            await store.setStage("SUBMITTING");

            // Step 3: Build payload
            const submitRows = await store.getSubmitRows();

            // Map rows to the backend's ImportRow format (grade_level + stream_name)
            const rows = submitRows.map((r) => ({
                full_name: r.processed_data.full_name,
                gender: r.processed_data.gender as "M" | "F",
                date_of_birth: r.processed_data.date_of_birth ?? undefined,
                upi_number: r.processed_data.nemis_number ?? undefined,
                knec_assessment_number: r.processed_data.assessment_number ?? undefined,
                admission_number: r.processed_data.birth_certificate_number ?? undefined,
                grade_level: r.processed_data.grade_level,
                stream_name: r.processed_data.stream_name,
            }));

            if (!store.meta) {
                throw new Error("Import session not initialized");
            }

            // Step 4: POST /students/import
            const response: ImportResponse = await submitStudentImport({
                idempotency_key: idempotencyKey,
                academic_term_id: store.meta.academic_term_id,
                rows,
            });

            // Step 5: Persist job_id immediately on receipt
            await store.setImportJobId(response.job_id);

            // Step 6: Transition to SUBMITTING
            onStageChange("SUBMITTING", response.job_id);
        } catch (err) {
            setError(getErrorMessage(err));
            // Revert to READY on failure
            await store.setStage("READY");
            setSubmitting(false);
        }
    };

    // ─── Render ───────────────────────────────────────────────────────────
    return (
        <div className="flex flex-1 flex-col gap-6 py-4">
            {error && (
                <div className="text-destructive bg-destructive/10 rounded-md px-3 py-2 text-sm">
                    {error}
                </div>
            )}

            {/* Summary card */}
            <div className="space-y-4">
                <p className="text-foreground text-sm font-medium">Ready to Import</p>

                <div className="grid grid-cols-3 gap-4">
                    <div className="bg-muted/30 rounded-lg p-4">
                        <p className="text-2xl font-semibold tracking-tight">{totalRows}</p>
                        <p className="text-muted-foreground text-xs">Total rows</p>
                    </div>
                    <div className="bg-muted/30 rounded-lg p-4">
                        <p className="text-primary text-2xl font-semibold tracking-tight">
                            {toSubmit}
                        </p>
                        <p className="text-muted-foreground text-xs">To be submitted</p>
                    </div>
                    <div className="bg-muted/30 rounded-lg p-4">
                        <p className="text-2xl font-semibold tracking-tight">
                            {store.skippedCount}
                        </p>
                        <p className="text-muted-foreground text-xs">Skipped</p>
                    </div>
                </div>

                {store.meta?.file_name && (
                    <p className="text-muted-foreground text-xs">
                        Source file: {store.meta.file_name}
                    </p>
                )}
            </div>

            {/* Actions */}
            <div className="flex items-center justify-between border-t pt-4">
                <div className="space-y-1">
                    {hasErrors && (
                        <p className="text-destructive text-xs">
                            Resolve all errors before uploading
                        </p>
                    )}
                </div>
                <div className="flex items-center gap-2">
                    <Button variant="ghost" onClick={onClose}>
                        Cancel
                    </Button>
                    <Button
                        variant="ghost"
                        onClick={() => {
                            store.setStage("PREVIEW");
                            onStageChange("PREVIEW");
                        }}
                    >
                        Back
                    </Button>
                    <Button onClick={handleUpload} disabled={hasErrors || submitting}>
                        {submitting ? (
                            <>
                                <Loader2 className="mr-1.5 size-4 animate-spin" />
                                Submitting…
                            </>
                        ) : (
                            "Upload"
                        )}
                    </Button>
                </div>
            </div>
        </div>
    );
}
