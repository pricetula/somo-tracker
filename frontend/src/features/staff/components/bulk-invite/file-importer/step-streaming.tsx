"use client";

/**
 * StepStreaming — submit the validated invitation records.
 * Matches the student import StepStreaming pattern — reads from IndexedDB.
 */

import * as React from "react";
import { Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { getErrorMessage } from "@/lib/errors";
import { submitBulkInvite, getImportAlreadyInProgress } from "@/lib/api/invitations";
import { getStagedRecordsByStatus } from "./db";
import type { StagedInviteRecord } from "./types";

interface StepStreamingProps {
    onError: (error: string) => void;
    onJobCreated: (jobId: string, totalRecords: number) => void;
    schoolId: string;
    role: string;
}

// ─── Main Component ───────────────────────────────────────────────────────

export function StepStreaming({ onError, onJobCreated, schoolId, role }: StepStreamingProps) {
    const [records, setRecords] = React.useState<StagedInviteRecord[]>([]);
    const [loading, setLoading] = React.useState(true);
    const [submitting, setSubmitting] = React.useState(false);
    const idempotencyKeyRef = React.useRef<string | null>(null);

    // Load valid records on mount
    React.useEffect(() => {
        getStagedRecordsByStatus(schoolId, "valid").then((valid) => {
            setRecords(valid);
            setLoading(false);
        });
    }, [schoolId]);

    const handleStartImport = React.useCallback(async () => {
        if (records.length === 0) {
            onError("No valid records to invite.");
            return;
        }

        setSubmitting(true);

        if (!idempotencyKeyRef.current) {
            idempotencyKeyRef.current = crypto.randomUUID();
        }

        const inviteRows = records.map((r) => ({
            email: r.email.trim(),
            ...(r.full_name.trim() ? { full_name: r.full_name.trim() } : {}),
        }));

        try {
            const result = await submitBulkInvite({
                role,
                rows: inviteRows,
            });
            idempotencyKeyRef.current = null;
            onJobCreated(result.job_id, inviteRows.length);
        } catch (err) {
            const activeJobId = getImportAlreadyInProgress(err);
            if (activeJobId) {
                onJobCreated(activeJobId, inviteRows.length);
                return;
            }

            setSubmitting(false);
            onError(getErrorMessage(err));
        }
    }, [records, role, onError, onJobCreated]);

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
                <h3 className="font-medium">Ready to Send</h3>
                <p className="text-muted-foreground mt-1 text-xs">
                    {records.length} invitation{records.length !== 1 ? "s" : ""} will be sent in a
                    single request.
                </p>
            </div>

            <Button size="sm" onClick={handleStartImport} disabled={submitting}>
                {submitting ? (
                    <>
                        <Loader2 className="mr-1.5 size-3.5 animate-spin" /> Submitting…
                    </>
                ) : (
                    `Send ${records.length} Invitation${records.length !== 1 ? "s" : ""}`
                )}
            </Button>
        </div>
    );
}
