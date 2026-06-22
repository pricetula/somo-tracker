/**
 * Bulk Staff Import Dialog — the main entry point for the bulk invitation utility.
 *
 * Receives `role` prop from host page (e.g. /nurses/add → role="NURSE").
 * Manages two entry branches:
 *   1. Manual Multi-Add (inline repeating form rows)
 *   2. File Upload (drag-and-drop, CSV/XLSX parsed in Web Worker)
 *
 * Persists drafts to IndexedDB, validates via TanStack Virtual + RxJS.
 */

"use client";

import * as React from "react";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
} from "@/components/ui/dialog";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { FileUp, UserPlus } from "lucide-react";
import { toast } from "sonner";

import { ManualEntryPanel } from "./manual-entry-panel";
import { FileUploadPanel } from "./file-upload-panel";
import { ImportProgressPanel } from "./import-progress-panel";
import { CorrectionPanel } from "./correction-panel";

import { loadDraft, saveDraft, clearDraft, type ImportDraftRow } from "@/lib/db";

// ─── Types ─────────────────────────────────────────────────────────────────

export type AllowedRole = "SCHOOL_ADMIN" | "NURSE" | "FINANCE";

interface BulkStaffImportDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    role: AllowedRole;
    tenantID: string;
    userID: string;
}

type DialogView =
    | "entry" // Manual or file upload
    | "review" // Sample 5-row review before submit
    | "progress" // SSE progress tracking
    | "correction" // Post-import correction of failed rows
    | "done"; // Finished

const CONTEXT_PREFIX = "staff-import";

// ─── Component ─────────────────────────────────────────────────────────────

export function BulkStaffImportDialog({
    open,
    onOpenChange,
    role,
    tenantID,
    userID,
}: BulkStaffImportDialogProps) {
    const [view, setView] = React.useState<DialogView>("entry");
    const [rows, setRows] = React.useState<ImportDraftRow[]>([]);
    const [importJobID, setImportJobID] = React.useState<string | null>(null);
    const [hasResumePrompt, setHasResumePrompt] = React.useState<boolean>(false);

    const context = `${CONTEXT_PREFIX}:${role}`;

    // Uses event handler pattern (onOpenChange) rather than useEffect + setState
    function handleDialogOpen(isOpen: boolean) {
        if (!isOpen) {
            clearDraft(tenantID, userID, context);
            onOpenChange(false);
            return;
        }
        setView("entry");
        setImportJobID(null);
        setHasResumePrompt(false);
        setRows([]);

        loadDraft(tenantID, userID, context).then((draft) => {
            if (draft && draft.rows.length > 0) {
                setHasResumePrompt(true);
                setRows(draft.rows);
            }
        });
    }

    function handleResumeDraft() {
        setHasResumePrompt(false);
    }

    function handleClearDraft() {
        clearDraft(tenantID, userID, context);
        setRows([]);
        setHasResumePrompt(false);
    }

    function handleRowsReady(newRows: ImportDraftRow[]) {
        setRows(newRows);
        saveDraft(tenantID, userID, context, newRows);

        // Show 5-row sample review
        if (getCriticalErrorCount(newRows) === 0) {
            setView("review");
        }
    }

    function handleSubmit(jobID: string) {
        setImportJobID(jobID);
        clearDraft(tenantID, userID, context);
        setView("progress");
    }

    function handleProgressDone(jobID: string, hasErrors: boolean) {
        if (hasErrors) {
            setImportJobID(jobID);
            setView("correction");
        } else {
            toast("Import complete", {
                description: "All invitations sent successfully.",
            });
            setView("done");
            setTimeout(() => onOpenChange(false), 1500);
        }
    }

    function handleCorrectionDone() {
        toast("Corrections submitted");
        setView("done");
        setTimeout(() => onOpenChange(false), 1500);
    }

    const roleLabel =
        role === "SCHOOL_ADMIN" ? "School Admins" : role === "NURSE" ? "Nurses" : "Finance Staff";

    return (
        <Dialog open={open} onOpenChange={handleDialogOpen}>
            <DialogContent className="flex max-h-[90vh] flex-col sm:max-w-4xl">
                <DialogHeader>
                    <DialogTitle>Invite {roleLabel}</DialogTitle>
                    <DialogDescription>
                        Add staff members one by one, or upload a CSV/XLSX file. All invitees will
                        receive the role of <strong>{role.toLowerCase().replace(/_/g, " ")}</strong>
                        .
                    </DialogDescription>
                </DialogHeader>

                <div className="flex-1 overflow-hidden">
                    {view === "entry" && (
                        <EntryView
                            hasResumePrompt={hasResumePrompt}
                            onResume={handleResumeDraft}
                            onClear={handleClearDraft}
                            onRowsReady={handleRowsReady}
                            role={role}
                            tenantID={tenantID}
                            userID={userID}
                            context={context}
                        />
                    )}

                    {view === "review" && (
                        <ReviewView
                            rows={rows}
                            role={role}
                            onSubmit={handleSubmit}
                            onBack={() => setView("entry")}
                        />
                    )}

                    {view === "progress" && importJobID && (
                        <ImportProgressPanel
                            jobID={importJobID}
                            onDone={(hasErrors) => handleProgressDone(importJobID, hasErrors)}
                            onClose={() => onOpenChange(false)}
                        />
                    )}

                    {view === "correction" && importJobID && (
                        <CorrectionPanel
                            jobID={importJobID}
                            role={role}
                            tenantID={tenantID}
                            userID={userID}
                            onSubmit={handleCorrectionDone}
                            onClose={() => onOpenChange(false)}
                        />
                    )}

                    {view === "done" && (
                        <div className="text-muted-foreground flex items-center justify-center py-16">
                            <p>Complete. Closing...</p>
                        </div>
                    )}
                </div>
            </DialogContent>
        </Dialog>
    );
}

// ─── Sub-components ─────────────────────────────────────────────────────

interface EntryViewProps {
    hasResumePrompt: boolean;
    onResume: () => void;
    onClear: () => void;
    onRowsReady: (rows: ImportDraftRow[]) => void;
    role: AllowedRole;
    tenantID: string;
    userID: string;
    context: string;
}

function EntryView({
    hasResumePrompt,
    onResume,
    onClear,
    onRowsReady,
    role,
    tenantID,
    userID,
    context,
}: EntryViewProps) {
    if (hasResumePrompt) {
        return (
            <div className="flex flex-col items-center justify-center gap-4 py-16">
                <p className="text-muted-foreground text-sm">
                    You have an unfinished import draft. Would you like to resume it?
                </p>
                <div className="flex gap-3">
                    <button
                        onClick={onResume}
                        className="bg-primary text-primary-foreground hover:bg-primary/90 rounded-md px-4 py-2 text-sm font-medium"
                    >
                        Resume Draft
                    </button>
                    <button
                        onClick={onClear}
                        className="bg-secondary text-secondary-foreground hover:bg-secondary/80 rounded-md px-4 py-2 text-sm font-medium"
                    >
                        Start Fresh
                    </button>
                </div>
            </div>
        );
    }

    return (
        <Tabs defaultValue="manual" className="flex flex-col">
            <TabsList className="mx-auto mb-4">
                <TabsTrigger value="manual" className="gap-2">
                    <UserPlus className="size-4" />
                    Add Manually
                </TabsTrigger>
                <TabsTrigger value="upload" className="gap-2">
                    <FileUp className="size-4" />
                    Upload File
                </TabsTrigger>
            </TabsList>

            <TabsContent value="manual" className="mt-0 flex-1">
                <ManualEntryPanel
                    onRowsReady={onRowsReady}
                    role={role}
                    tenantID={tenantID}
                    userID={userID}
                    context={context}
                />
            </TabsContent>

            <TabsContent value="upload" className="mt-0 flex-1">
                <FileUploadPanel
                    onRowsReady={onRowsReady}
                    role={role}
                    tenantID={tenantID}
                    userID={userID}
                    context={context}
                />
            </TabsContent>
        </Tabs>
    );
}

// ─── Review View ──────────────────────────────────────────────────────────

interface ReviewViewProps {
    rows: ImportDraftRow[];
    role: AllowedRole;
    onSubmit: (jobID: string) => void;
    onBack: () => void;
}

function ReviewView({ rows, role, onSubmit, onBack }: ReviewViewProps) {
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

            <div className="bg-muted/30 overflow-hidden rounded-lg border">
                <table className="w-full text-left text-sm">
                    <thead>
                        <tr className="bg-muted/50 border-b">
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

// ─── Virtualized Grid ─────────────────────────────────────────────────────

import { useVirtualizer } from "@tanstack/react-virtual";
import { isValidPhoneNumber, parsePhoneNumber } from "libphonenumber-js";

// Re-export for sub-components
export { useVirtualizer, isValidPhoneNumber, parsePhoneNumber };

/** Check if email has valid '@' structural layout. */
export function hasValidEmailStructure(email: string): boolean {
    if (!email || !email.includes("@")) return false;
    const parts = email.split("@");
    if (parts.length !== 2) return false;
    const [local, domain] = parts;
    if (local.length === 0 || domain.length === 0) return false;
    if (!domain.includes(".")) return false;
    return true;
}

/** Normalize phone to E.164 with default country KE. Returns null if unparseable. */
export function normalizePhone(phone: string): string | null {
    if (!phone || phone.trim() === "") return null;
    const cleaned = phone.trim();
    try {
        if (isValidPhoneNumber(cleaned, "KE")) {
            return parsePhoneNumber(cleaned, "KE")!.format("E.164");
        }
        // Try parsing anyway
        const parsed = parsePhoneNumber(cleaned, "KE");
        if (parsed && isValidPhoneNumber(cleaned, "KE")) {
            return parsed.format("E.164");
        }
        return null;
    } catch {
        return null;
    }
}

/** Count critical errors (block submission). */
export function getCriticalErrorCount(rows: ImportDraftRow[]): number {
    let errors = 0;
    const emails = new Set<string>();
    for (const row of rows) {
        if (!row.first_name) errors++;
        if (!row.last_name) errors++;
        if (!hasValidEmailStructure(row.email)) errors++;
        const lowerEmail = row.email.toLowerCase();
        if (emails.has(lowerEmail)) errors++;
        emails.add(lowerEmail);
    }
    return errors;
}

// ─── Re-exports from hooks ──────────────────────────────────────────────

import { useStartImport } from "../hooks/use-staff-import";
