/**
 * Review View — shows a 5-row sample preview before submitting the batch import.
 *
 * Displays the first 5 records for visual confirmation, then triggers the
 * import mutation on submit.
 */

"use client";

import * as React from "react";

import { useStartImport } from "../hooks/use-staff-import";
import type { ImportDraftRow } from "@/lib/db";
import type { AllowedRole } from "./bulk-staff-import-dialog";

// ─── Types ─────────────────────────────────────────────────────────────────

export interface ReviewViewProps {
    rows: ImportDraftRow[];
    role: AllowedRole;
    onSubmit: (jobID: string) => void;
    onBack: () => void;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function ReviewView({ rows, role, onSubmit, onBack }: ReviewViewProps) {
    const [sending, setSending] = React.useState(false);
    const startImportMutation = useStartImport();

    const sampleRows = rows.slice(0, 5);

    async function handleConfirm() {
        if (sending) return;
        setSending(true);
        try {
            const result = await startImportMutation.mutateAsync({
                role,
                records: rows.map((r) => ({
                    temp_id: r.temp_id,
                    email: r.email,
                    first_name: r.first_name,
                    last_name: r.last_name,
                    phone: r.phone,
                    registration_number: r.registration_number,
                })),
            });
            onSubmit(result.import_job_id);
        } catch {
            // Handled by mutation
        } finally {
            setSending(false);
        }
    }

    return (
        <div className="flex flex-col gap-4 p-4">
            <p className="text-muted-foreground text-sm">
                All {rows.length} record{rows.length !== 1 ? "s" : ""} are valid. Please review the
                first 5 rows below before submitting.
            </p>

            <div className="bg-muted/30 max-h-72 overflow-auto rounded-lg border">
                <table className="w-full text-left text-sm">
                    <thead className="bg-muted/50 sticky top-0">
                        <tr className="border-b">
                            <th className="px-3 py-2 font-medium">Email</th>
                            <th className="px-3 py-2 font-medium">First Name</th>
                            <th className="px-3 py-2 font-medium">Last Name</th>
                            <th className="px-3 py-2 font-medium">Phone</th>
                        </tr>
                    </thead>
                    <tbody>
                        {sampleRows.map((row) => (
                            <tr key={row.temp_id} className="border-b last:border-0">
                                <td className="px-3 py-2">{row.email}</td>
                                <td className="px-3 py-2">{row.first_name}</td>
                                <td className="px-3 py-2">{row.last_name}</td>
                                <td className="px-3 py-2">{row.phone || "—"}</td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>

            <div className="flex items-center justify-between">
                <p className="text-muted-foreground text-xs">
                    {rows.length > 5 && `+ ${rows.length - 5} more records`}
                </p>
                <div className="flex gap-3">
                    <button
                        onClick={onBack}
                        className="text-muted-foreground hover:text-foreground rounded-md px-4 py-2 text-sm font-medium"
                        disabled={sending}
                    >
                        Back
                    </button>
                    <button
                        onClick={handleConfirm}
                        disabled={sending}
                        className="bg-primary text-primary-foreground hover:bg-primary/90 rounded-md px-6 py-2 text-sm font-medium disabled:opacity-50"
                    >
                        {sending
                            ? "Submitting..."
                            : `Submit ${rows.length} Invitation${rows.length !== 1 ? "s" : ""}`}
                    </button>
                </div>
            </div>
        </div>
    );
}
