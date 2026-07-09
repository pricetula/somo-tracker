/**
 * BulkInviteManualForm — invite staff by typing email addresses in a table.
 *
 * Pattern matches StudentManualImportForm:
 *   - Add/remove rows
 *   - Within-batch duplicate detection
 *   - POST all rows at once → delegate to ImportProgress
 */

"use client";

import * as React from "react";
import { Plus, Trash2, Loader2, AlertCircle, Send } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { submitBulkInvite, getImportAlreadyInProgress } from "@/lib/api/invitations";
import { getErrorMessage } from "@/lib/errors";
import { validateInviteRow, detectDuplicateEmails } from "./validation-utils";
import type { InviteRowInput, InviteRowError } from "./validation-utils";

// ─── Constants ────────────────────────────────────────────────────────────

// MAX_INVITE_ROWS must stay in sync with backend imports.MaxImportRows (5000).
const MAX_INVITE_ROWS = 5000;

// ─── Row ID counter ───────────────────────────────────────────────────────

let rowCounter = 0;

interface InviteFormRow {
    id: string;
    email: string;
    fullName: string;
}

function freshRow(): InviteFormRow {
    rowCounter += 1;
    return {
        id: `invite-${rowCounter}-${Date.now()}`,
        email: "",
        fullName: "",
    };
}

// ─── Props ────────────────────────────────────────────────────────────────

interface BulkInviteManualFormProps {
    role: string;
    onReset: () => void;
    onJobCreated: (jobId: string, totalRecords: number) => void;
}

// ─── Component ────────────────────────────────────────────────────────────

export function BulkInviteManualForm({ role, onReset, onJobCreated }: BulkInviteManualFormProps) {
    const [rows, setRows] = React.useState<InviteFormRow[]>([freshRow()]);
    const [submitting, setSubmitting] = React.useState(false);
    const submittingRef = React.useRef(false);

    // ── Validation (on every render) ───────────────────────────────────
    const inputRows: InviteRowInput[] = React.useMemo(
        () => rows.map((r) => ({ email: r.email, full_name: r.fullName })),
        [rows]
    );

    const rowErrors = React.useMemo(() => {
        const errors: InviteRowError[] = [];
        inputRows.forEach((row, i) => {
            errors.push(...validateInviteRow(row, i));
        });
        errors.push(...detectDuplicateEmails(inputRows));
        return errors;
    }, [inputRows]);

    const hasErrors = rowErrors.length > 0;
    const nonEmptyRows = rows.filter((r) => r.email.trim().length > 0);
    const canSubmit = nonEmptyRows.length > 0 && !hasErrors && !submitting;

    function getRowError(rowId: string, field: "email" | "full_name"): string | undefined {
        const idx = rows.findIndex((r) => r.id === rowId);
        if (idx === -1) return undefined;
        return rowErrors.find((e) => e.rowIndex === idx && e.field === field)?.message;
    }

    // ── Row mutation ───────────────────────────────────────────────────
    function updateRow(rowId: string, patch: Partial<InviteFormRow>) {
        setRows((prev) => prev.map((r) => (r.id === rowId ? { ...r, ...patch } : r)));
    }

    function removeRow(rowId: string) {
        setRows((prev) => {
            if (prev.length <= 1) return [freshRow()];
            return prev.filter((r) => r.id !== rowId);
        });
    }

    function addRow() {
        setRows((prev) => {
            if (prev.length >= MAX_INVITE_ROWS) {
                toast.error(`Maximum of ${MAX_INVITE_ROWS.toLocaleString()} rows reached.`);
                return prev;
            }
            return [...prev, freshRow()];
        });
    }

    // ── Submit ─────────────────────────────────────────────────────────
    async function handleSubmit() {
        if (submittingRef.current || !canSubmit) return;

        submittingRef.current = true;
        setSubmitting(true);

        const inviteRows = nonEmptyRows.map((r) => ({
            email: r.email.trim(),
            ...(r.fullName.trim() ? { full_name: r.fullName.trim() } : {}),
        }));

        try {
            const result = await submitBulkInvite({ role, rows: inviteRows });
            onJobCreated(result.job_id, inviteRows.length);
        } catch (err) {
            const activeJobId = getImportAlreadyInProgress(err);
            if (activeJobId) {
                onJobCreated(activeJobId, inviteRows.length);
                return;
            }
            submittingRef.current = false;
            setSubmitting(false);
            toast.error(getErrorMessage(err));
        }
    }

    // ── Render ─────────────────────────────────────────────────────────
    return (
        <div className="space-y-4">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div>
                    <h3 className="text-sm font-medium">Manual Entry</h3>
                    {hasErrors && (
                        <p className="text-destructive mt-0.5 text-xs">
                            {rowErrors.length} error{rowErrors.length !== 1 ? "s" : ""}
                        </p>
                    )}
                </div>
                <div className="flex items-center gap-2">
                    <Button variant="outline" size="sm" onClick={onReset} disabled={submitting}>
                        Cancel
                    </Button>
                </div>
            </div>

            {/* Rows */}
            {rows.length === 0 ? (
                <div className="text-muted-foreground flex flex-col items-center gap-3 py-12 text-xs">
                    <p>No emails added yet.</p>
                    <Button variant="outline" size="sm" onClick={addRow}>
                        <Plus className="mr-1 size-3.5" />
                        Add an email
                    </Button>
                </div>
            ) : (
                <div className="space-y-2">
                    {rows.map((row, index) => {
                        const emailErr = getRowError(row.id, "email");
                        const hasRowError = !!emailErr;

                        return (
                            <div
                                key={row.id}
                                className={`rounded-md p-3 ${
                                    hasRowError
                                        ? "bg-destructive/5 ring-destructive/20 ring-1"
                                        : "bg-muted/30"
                                }`}
                            >
                                <div className="mb-2 flex items-center justify-between">
                                    <span className="text-muted-foreground text-xs font-medium">
                                        Person {index + 1}
                                        {hasRowError && (
                                            <span className="text-destructive ml-1">(error)</span>
                                        )}
                                    </span>
                                    <Button
                                        variant="ghost"
                                        size="icon-xs"
                                        onClick={() => removeRow(row.id)}
                                        disabled={submitting || rows.length === 1}
                                        className="text-muted-foreground hover:text-destructive size-6"
                                        aria-label={`Remove row ${index + 1}`}
                                    >
                                        <Trash2 className="size-3.5" />
                                    </Button>
                                </div>

                                <div className="flex flex-col gap-2 sm:flex-row sm:items-start">
                                    {/* Email */}
                                    <div className="flex-1 space-y-1">
                                        <Label
                                            className="text-[0.625rem]"
                                            htmlFor={`email-${row.id}`}
                                        >
                                            Email <span className="text-destructive">*</span>
                                        </Label>
                                        <Input
                                            id={`email-${row.id}`}
                                            value={row.email}
                                            onChange={(e) =>
                                                updateRow(row.id, { email: e.target.value })
                                            }
                                            placeholder="teacher@school.com"
                                            disabled={submitting}
                                            className={`h-8 text-xs ${
                                                emailErr ? "border-destructive" : ""
                                            }`}
                                        />
                                        {emailErr && (
                                            <p className="text-destructive mt-0.5 flex items-center gap-1 text-[0.625rem]">
                                                <AlertCircle className="size-3" />
                                                {emailErr}
                                            </p>
                                        )}
                                    </div>

                                    {/* Full Name */}
                                    <div className="flex-1 space-y-1">
                                        <Label
                                            className="text-[0.625rem]"
                                            htmlFor={`name-${row.id}`}
                                        >
                                            Full Name
                                        </Label>
                                        <Input
                                            id={`name-${row.id}`}
                                            value={row.fullName}
                                            onChange={(e) =>
                                                updateRow(row.id, { fullName: e.target.value })
                                            }
                                            placeholder="Jane Doe (optional)"
                                            disabled={submitting}
                                            className="h-8 text-xs"
                                        />
                                    </div>
                                </div>
                            </div>
                        );
                    })}
                </div>
            )}

            {/* Footer */}
            {rows.length > 0 && (
                <div className="flex items-center justify-between pt-1">
                    <div className="text-muted-foreground text-[0.625rem]">
                        {nonEmptyRows.length} of {rows.length} row
                        {rows.length !== 1 ? "s" : ""} ready
                        {hasErrors && (
                            <span className="text-destructive ml-1">
                                · {rowErrors.length} error{rowErrors.length !== 1 ? "s" : ""}
                            </span>
                        )}
                    </div>
                    <div className="flex items-center gap-2">
                        <Button
                            variant="outline"
                            size="sm"
                            onClick={addRow}
                            disabled={submitting || rows.length >= MAX_INVITE_ROWS}
                        >
                            <Plus className="mr-1 size-3.5" /> Add Row
                        </Button>
                        <Button size="sm" onClick={handleSubmit} disabled={!canSubmit}>
                            {submitting ? (
                                <>
                                    <Loader2 className="mr-1.5 size-3.5 animate-spin" />
                                    Sending…
                                </>
                            ) : (
                                <>
                                    <Send className="mr-1.5 size-3.5" />
                                    Invite {nonEmptyRows.length}
                                </>
                            )}
                        </Button>
                    </div>
                </div>
            )}
        </div>
    );
}
