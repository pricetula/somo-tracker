/**
 * Bulk Staff Import — the main entry point for the bulk invitation utility.
 *
 * Composable, mode-aware component:
 *   - mode='dialog': renders just the form body (caller provides the modal shell)
 *   - mode='page': renders the full form with inline success state
 *
 * Receives `role` prop from host page (e.g. /nurses/invitations → role="NURSE").
 * Resolves tenant_id and user_id internally from the session.
 *
 * The component never calls router.back() or router.push() — that is the
 * caller's responsibility via onClose and onSuccess.
 */

"use client";

import * as React from "react";
import { toast } from "sonner";

import { EntryView } from "./entry-view";
import { ReviewView } from "./review-view";
import { ImportProgressPanel } from "./import-progress-panel";
import { CorrectionPanel } from "./correction-panel";

import { loadDraft, saveDraft, clearDraft, type ImportDraftRow } from "@/lib/db";
import { getMe } from "@/lib/api/auth";

import { getCriticalErrorCount } from "../lib/validation";

// ─── Types ─────────────────────────────────────────────────────────────────

export type AllowedRole = "SCHOOL_ADMIN" | "NURSE" | "FINANCE" | "TEACHER";

export interface BulkStaffImportProps {
    role: AllowedRole;
    mode: "dialog" | "page";
    onSuccess?: () => void;
    onClose?: () => void;
}

type DialogView =
    | "entry" // Manual or file upload
    | "review" // Sample 5-row review before submit
    | "progress" // SSE progress tracking
    | "correction" // Post-import correction of failed rows
    | "done"; // Finished

const CONTEXT_PREFIX = "staff-import";

// ─── Component ─────────────────────────────────────────────────────────────

export function BulkStaffImport({ role, mode, onSuccess, onClose }: BulkStaffImportProps) {
    const [view, setView] = React.useState<DialogView>("entry");
    const [rows, setRows] = React.useState<ImportDraftRow[]>([]);
    const [importJobID, setImportJobID] = React.useState<string | null>(null);
    const [hasResumePrompt, setHasResumePrompt] = React.useState<boolean>(false);
    const [tenantID, setTenantID] = React.useState<string | null>(null);
    const [userID, setUserID] = React.useState<string | null>(null);
    const [sessionLoading, setSessionLoading] = React.useState(true);

    const context = `${CONTEXT_PREFIX}:${role}`;

    // Resolve tenant_id and user_id from the session on mount
    React.useEffect(() => {
        let cancelled = false;
        getMe()
            .then((me) => {
                if (!cancelled) {
                    setTenantID(me.tenant_id);
                    setUserID(me.user_id);
                    setSessionLoading(false);

                    // Load draft after we have the session
                    loadDraft(me.tenant_id, me.user_id, context).then((draft) => {
                        if (!cancelled && draft && draft.rows.length > 0) {
                            setHasResumePrompt(true);
                            setRows(draft.rows);
                        }
                    });
                }
            })
            .catch(() => {
                if (!cancelled) {
                    setSessionLoading(false);
                }
            });
        return () => {
            cancelled = true;
        };
    }, [context]);

    function handleResumeDraft() {
        setHasResumePrompt(false);
    }

    function handleClearDraft() {
        if (tenantID && userID) {
            clearDraft(tenantID, userID, context);
        }
        setRows([]);
        setHasResumePrompt(false);
    }

    function handleRowsReady(newRows: ImportDraftRow[]) {
        setRows(newRows);
        if (tenantID && userID) {
            saveDraft(tenantID, userID, context, newRows);
        }

        // Show 5-row sample review
        if (getCriticalErrorCount(newRows) === 0) {
            setView("review");
        }
    }

    function handleSubmit(jobID: string) {
        setImportJobID(jobID);
        if (tenantID && userID) {
            clearDraft(tenantID, userID, context);
        }
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
            if (mode === "dialog") {
                setView("done");
                setTimeout(() => onClose?.(), 1500);
            } else {
                onSuccess?.();
                setView("done");
            }
        }
    }

    function handleCorrectionDone() {
        toast("Corrections submitted");
        if (mode === "dialog") {
            setView("done");
            setTimeout(() => onClose?.(), 1500);
        } else {
            onSuccess?.();
            setView("done");
        }
    }

    const roleLabel =
        role === "SCHOOL_ADMIN"
            ? "School Admins"
            : role === "TEACHER"
              ? "Teachers"
              : role === "NURSE"
                ? "Nurses"
                : "Finance Staff";

    if (sessionLoading) {
        return (
            <div className="flex items-center justify-center py-16">
                <p className="text-muted-foreground text-sm">Loading...</p>
            </div>
        );
    }

    const body = (
        <div className="flex flex-1 flex-col overflow-hidden">
            <div className="mb-4">
                <h2 className="text-lg font-semibold">Invite {roleLabel}</h2>
                <p className="text-muted-foreground mt-1 text-sm">
                    Add staff members one by one, or upload a CSV/XLSX file. All invitees will
                    receive the role of <strong>{role.toLowerCase().replace(/_/g, " ")}</strong>.
                </p>
            </div>

            <div className="flex-1 overflow-hidden">
                {view === "entry" && (
                    <EntryView
                        hasResumePrompt={hasResumePrompt}
                        onResume={handleResumeDraft}
                        onClear={handleClearDraft}
                        onRowsReady={handleRowsReady}
                        role={role}
                        tenantID={tenantID ?? ""}
                        userID={userID ?? ""}
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
                        onClose={() => onClose?.()}
                    />
                )}

                {view === "correction" && importJobID && tenantID && userID && (
                    <CorrectionPanel
                        jobID={importJobID}
                        role={role}
                        tenantID={tenantID}
                        userID={userID}
                        onSubmit={handleCorrectionDone}
                        onClose={() => onClose?.()}
                    />
                )}

                {view === "done" && (
                    <div className="text-muted-foreground flex items-center justify-center py-16">
                        <div className="text-center">
                            <p className="text-sm font-medium">Import Complete</p>
                            <p className="mt-1 text-xs">
                                All invitations have been processed successfully.
                            </p>
                        </div>
                    </div>
                )}
            </div>
        </div>
    );

    // In dialog mode, render just the body — the caller wraps it in a modal shell
    if (mode === "dialog") {
        return body;
    }

    // In page mode, render with a full-page wrapper
    return <div className="mx-auto max-w-4xl px-6 py-6">{body}</div>;
}
