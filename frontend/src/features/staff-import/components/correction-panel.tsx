/**
 * Correction Panel — post-import recovery and resolution.
 *
 * Loads failed invitations for a completed job and presents them
 * in an editable grid. Resubmitting creates a new import_job with
 * parent_import_job_id linking back to the original.
 */

"use client";

import * as React from "react";
import { useVirtualizer } from "@tanstack/react-virtual";
import { AlertCircle, RefreshCw } from "lucide-react";

import { useImportFailures, useStartImport } from "../hooks/use-staff-import";
import type { AllowedRole } from "./bulk-staff-import-dialog";
import type { FailedInvitation } from "@/lib/api/imports";

// ─── Types ─────────────────────────────────────────────────────────────────

interface CorrectionPanelProps {
    jobID: string;
    role: AllowedRole;
    tenantID: string;
    userID: string;
    onSubmit: () => void;
    onClose: () => void;
}

interface CorrectedRow {
    id: string;
    email: string;
    first_name: string;
    last_name: string;
    phone: string;
    registration_number: string;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function CorrectionPanel({ jobID, role, onSubmit, onClose }: CorrectionPanelProps) {
    const parentRef = React.useRef<HTMLDivElement>(null);
    const [rows, setRows] = React.useState<CorrectedRow[]>([]);
    const [sending, setSending] = React.useState(false);

    const { data, isLoading } = useImportFailures(jobID);

    React.useEffect(() => {
        if (data?.invitations) {
            setRows(
                data.invitations.map((inv: FailedInvitation) => ({
                    id: inv.id,
                    email: inv.email,
                    first_name: inv.first_name ?? "",
                    last_name: inv.last_name ?? "",
                    phone: inv.phone ?? "",
                    registration_number: "",
                }))
            );
        }
    }, [data]);

    // eslint-disable-next-line react-hooks/incompatible-library
    const virtualizer = useVirtualizer({
        count: rows.length,
        getScrollElement: () => parentRef.current,
        estimateSize: () => 48,
        overscan: 5,
    });

    function updateRow(id: string, field: keyof CorrectedRow, value: string) {
        setRows((prev) => prev.map((r) => (r.id === id ? { ...r, [field]: value } : r)));
    }

    const startImportMutation = useStartImport();

    async function handleResubmit() {
        if (sending || rows.length === 0) return;
        setSending(true);

        try {
            await startImportMutation.mutateAsync({
                role,
                records: rows.map((r) => ({
                    temp_id: r.id,
                    email: r.email,
                    first_name: r.first_name,
                    last_name: r.last_name,
                    phone: r.phone,
                    registration_number: r.registration_number,
                })),
            });
            onSubmit();
        } catch {
            // Handled by mutation
        } finally {
            setSending(false);
        }
    }

    if (isLoading) {
        return (
            <div className="text-muted-foreground flex items-center justify-center py-12 text-sm">
                Loading failed records...
            </div>
        );
    }

    if (!data) {
        return (
            <div className="text-destructive flex items-center justify-center gap-2 py-12 text-sm">
                <AlertCircle className="size-4" />
                Failed to load error records
            </div>
        );
    }

    if (rows.length === 0) {
        return (
            <div className="flex flex-col items-center justify-center gap-3 py-12">
                <p className="text-muted-foreground text-sm">No failed records found.</p>
                <button onClick={onClose} className="text-primary text-sm underline">
                    Close
                </button>
            </div>
        );
    }

    return (
        <div className="flex flex-col gap-3">
            <div className="bg-destructive/10 text-destructive flex items-start gap-2 rounded-md px-3 py-2 text-sm">
                <AlertCircle className="mt-0.5 size-4 shrink-0" />
                <span>
                    {rows.length} invitation{rows.length !== 1 ? "s" : ""} failed to send. Correct
                    the issues below and resubmit.
                </span>
            </div>

            {/* Column headers */}
            <div
                className="text-muted-foreground grid gap-2 px-1 text-xs font-medium"
                style={{
                    gridTemplateColumns:
                        role === "TEACHER" ? "1.5fr 1fr 1fr 0.8fr 1fr" : "1.5fr 1fr 1fr 1fr",
                }}
            >
                <span>Email *</span>
                <span>First Name</span>
                <span>Last Name</span>
                {role === "TEACHER" && <span>TSC Number</span>}
                <span>Phone</span>
            </div>

            {/* Virtualized editable rows */}
            <div ref={parentRef} className="max-h-87.5 overflow-auto">
                <div
                    style={{
                        height: `${virtualizer.getTotalSize()}px`,
                        width: "100%",
                        position: "relative",
                    }}
                >
                    {virtualizer.getVirtualItems().map((virtualItem) => {
                        const row = rows[virtualItem.index];
                        return (
                            <div
                                key={row.id}
                                data-index={virtualItem.index}
                                ref={virtualizer.measureElement}
                                className="absolute top-0 left-0 w-full"
                                style={{
                                    transform: `translateY(${virtualItem.start}px)`,
                                }}
                            >
                                <div
                                    className="grid gap-2 px-1 py-1"
                                    style={{
                                        gridTemplateColumns:
                                            role === "TEACHER"
                                                ? "1.5fr 1fr 1fr 0.8fr 1fr"
                                                : "1.5fr 1fr 1fr 1fr",
                                    }}
                                >
                                    <input
                                        type="text"
                                        value={row.email}
                                        onChange={(e) => updateRow(row.id, "email", e.target.value)}
                                        className="border-input bg-background h-9 w-full rounded-md border px-3 text-sm"
                                    />
                                    <input
                                        type="text"
                                        value={row.first_name}
                                        onChange={(e) =>
                                            updateRow(row.id, "first_name", e.target.value)
                                        }
                                        className="border-input bg-background h-9 w-full rounded-md border px-3 text-sm"
                                    />
                                    <input
                                        type="text"
                                        value={row.last_name}
                                        onChange={(e) =>
                                            updateRow(row.id, "last_name", e.target.value)
                                        }
                                        className="border-input bg-background h-9 w-full rounded-md border px-3 text-sm"
                                    />
                                    {role === "TEACHER" && (
                                        <input
                                            type="text"
                                            value={row.registration_number}
                                            onChange={(e) =>
                                                updateRow(
                                                    row.id,
                                                    "registration_number",
                                                    e.target.value
                                                )
                                            }
                                            className="border-input bg-background h-9 w-full rounded-md border px-3 text-sm"
                                            placeholder="TSC-XXX-XXX"
                                        />
                                    )}
                                    <input
                                        type="text"
                                        value={row.phone}
                                        onChange={(e) => updateRow(row.id, "phone", e.target.value)}
                                        className="border-input bg-background h-9 w-full rounded-md border px-3 text-sm"
                                    />
                                </div>
                            </div>
                        );
                    })}
                </div>
            </div>

            {/* Actions */}
            <div className="flex items-center justify-between px-1 pt-2">
                <button
                    onClick={onClose}
                    className="text-muted-foreground hover:text-foreground rounded-md px-3 py-1.5 text-sm"
                    disabled={sending}
                >
                    Cancel
                </button>
                <button
                    onClick={handleResubmit}
                    disabled={sending || rows.length === 0}
                    className="bg-primary text-primary-foreground hover:bg-primary/90 flex items-center gap-2 rounded-md px-4 py-1.5 text-sm font-medium disabled:opacity-50"
                >
                    <RefreshCw className={`size-4 ${sending ? "animate-spin" : ""}`} />
                    {sending
                        ? "Resubmitting..."
                        : `Resubmit ${rows.length} Correction${rows.length !== 1 ? "s" : ""}`}
                </button>
            </div>
        </div>
    );
}
