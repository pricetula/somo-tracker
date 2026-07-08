"use client";

import * as React from "react";
import { Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { getErrorMessage } from "@/lib/errors";
import { submitStudentImport, type ImportRow } from "@/lib/api/imports";
import { getStagedRecordsByStatus } from "./db";
import type { StagedStudentRecord } from "./types";

interface StepStreamingProps {
    onComplete: () => void;
    onError: (error: string) => void;
    onJobCreated: (jobId: string, totalRecords: number) => void;
}

// ─── Helper: Build ImportRow from StagedStudentRecord ─────────────────────

function toImportRow(record: StagedStudentRecord): ImportRow {
    return {
        full_name: record.payload.full_name,
        gender: (record.payload.gender as "M" | "F") ?? "M",
        date_of_birth: record.payload.date_of_birth ?? null,
        upi_number: record.payload.upi_number ?? null,
        knec_assessment_number: record.payload.knec_assessment_number ?? null,
        class_id: record.payload.class_id ?? undefined,
    };
}

// ─── Main Component ───────────────────────────────────────────────────────

export function StepStreaming({ onError, onJobCreated }: StepStreamingProps) {
    const [records, setRecords] = React.useState<StagedStudentRecord[]>([]);
    const [loading, setLoading] = React.useState(true);
    const [submitting, setSubmitting] = React.useState(false);

    // Load valid records on mount
    React.useEffect(() => {
        getStagedRecordsByStatus("valid").then((valid) => {
            setRecords(valid);
            setLoading(false);
        });
    }, []);

    const handleStartImport = React.useCallback(async () => {
        if (records.length === 0) {
            onError("No valid records to import.");
            return;
        }

        setSubmitting(true);
        const rows: ImportRow[] = records.map(toImportRow);

        try {
            const result = await submitStudentImport({ rows });
            onJobCreated(result.job_id, rows.length);
        } catch (err) {
            setSubmitting(false);
            onError(getErrorMessage(err));
        }
    }, [records, onError, onJobCreated]);

    if (loading) {
        return (
            <div className="flex items-center justify-center py-8">
                <Loader2 className="text-muted-foreground size-5 animate-spin" />
            </div>
        );
    }

    return (
        <div className="space-y-4">
            <div>
                <h3 className="text-sm font-medium">Ready to Import</h3>
                <p className="text-muted-foreground mt-1 text-xs">
                    {records.length} student record{records.length !== 1 ? "s" : ""} will be sent in
                    a single request.
                </p>
            </div>

            <Button size="sm" onClick={handleStartImport} disabled={submitting}>
                {submitting ? (
                    <>
                        <Loader2 className="mr-1.5 size-3.5 animate-spin" /> Submitting…
                    </>
                ) : (
                    `Import ${records.length} Student${records.length !== 1 ? "s" : ""}`
                )}
            </Button>
        </div>
    );
}
