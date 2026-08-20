/**
 * BulkInviteForm — parent orchestrator for inviting staff members.
 *
 * Follows the same pattern as StudentsImportForm:
 *   - Active-job check on mount (GET /api/v1/imports/active)
 *   - Selector: "Manual Entry" or "Import File"
 *   - Manual: type emails in a table
 *   - File: upload CSV/Excel → map columns → review → submit
 *   - Shared ImportProgress for async progress tracking
 *   - One-active-job-per-school enforced by backend
 */

"use client";

import * as React from "react";
import { Loader2 } from "lucide-react";
import { ImportProgress } from "./import-progress";
import { getActiveImportJob, type ImportResponse } from "@/lib/api/imports";
import { useMe } from "@/hooks/use-auth";
import { BulkInviteSelector } from "./bulk-invite-selector";
import { BulkInviteManualForm } from "./bulk-invite-manual-form";
import { BulkInviteFileImporter } from "./bulk-invite-file-importer";

// ─── State machine ────────────────────────────────────────────────────────

type ImportStep = "selector" | "manual" | "file";

interface ActiveJob {
    jobId: string;
    totalRecords: number;
}

type PageState =
    | { phase: "active-job"; job: ActiveJob }
    | { phase: "idle"; step: ImportStep | null };

type PageAction =
    | { type: "SET_ACTIVE_JOB"; job: ActiveJob }
    | { type: "JOB_CREATED"; job: ActiveJob }
    | { type: "SELECT_STEP"; step: ImportStep }
    | { type: "RESET" };

function pageReducer(state: PageState, action: PageAction): PageState {
    switch (action.type) {
        case "SET_ACTIVE_JOB":
        case "JOB_CREATED":
            return { phase: "active-job", job: action.job };
        case "SELECT_STEP":
            return { phase: "idle", step: action.step };
        case "RESET":
            return { phase: "idle", step: null };
        default:
            return state;
    }
}

// ─── Props ────────────────────────────────────────────────────────────────

interface BulkInviteSubmitFn {
    (body: {
        role: string;
        rows: Array<{ email: string; full_name?: string }>;
    }): Promise<ImportResponse>;
}

interface BulkInviteFormProps {
    /** The role to pre-select. Determines who gets invited. */
    role: "SCHOOL_ADMIN" | "TEACHER" | "NURSE" | "FINANCE" | "PARENT";
    /** Custom submit function for different invite endpoints (e.g., parents vs staff). */
    submitFn?: BulkInviteSubmitFn;
    onSuccess: () => void;
}

// ─── Component ────────────────────────────────────────────────────────────

// ─── Role label map ──────────────────────────────────────────────────

const ROLE_LABELS: Record<string, string> = {
    TEACHER: "Invite Teachers",
    SCHOOL_ADMIN: "Invite Admins",
    NURSE: "Invite Nurses",
    FINANCE: "Invite Finance Staff",
    PARENT: "Invite Parents",
};

// ─── Component ────────────────────────────────────────────────────────────

export function BulkInviteForm({ role, submitFn, onSuccess }: BulkInviteFormProps) {
    const { data: me } = useMe();
    const [pageState, dispatch] = React.useReducer(pageReducer, { phase: "idle", step: null });

    const meResolved = me !== undefined;

    // Fire active-job check once when me resolves.
    // The active school is resolved from the authenticated session on the backend.
    React.useEffect(() => {
        if (!meResolved) return;

        let cancelled = false;

        getActiveImportJob()
            .then((result) => {
                if (cancelled) return;
                if (result.active && result.job) {
                    dispatch({
                        type: "SET_ACTIVE_JOB",
                        job: {
                            jobId: result.job.id,
                            totalRecords: result.job.total_records,
                        },
                    });
                }
            })
            .catch((err) => {
                // Transient — fall through to normal flow, but surface the failure
                // so an intermittent backend hiccup isn't silently swallowed.
                console.warn("Active import job check failed; assuming none is active.", err);
            });

        return () => {
            cancelled = true;
        };
    }, [meResolved]);

    // ── Handlers ──────────────────────────────────────────────────────

    function handleReset() {
        dispatch({ type: "RESET" });
        onSuccess();
    }

    function handleJobCreated(jobId: string, totalRecords: number) {
        dispatch({ type: "JOB_CREATED", job: { jobId, totalRecords } });
    }

    function handleRetry() {
        // dispatch({ type: "RESET" });
    }

    // ── Render ────────────────────────────────────────────────────────

    if (!meResolved) {
        return (
            <div className="flex items-center justify-center gap-2 py-12">
                <Loader2 className="text-muted-foreground size-5 animate-spin" />
                <p className="text-muted-foreground text-xs">Loading session...</p>
            </div>
        );
    }

    // Show active job progress regardless of selected step
    if (pageState.phase === "active-job") {
        return (
            <ImportProgress
                jobId={pageState.job.jobId}
                totalRecords={pageState.job.totalRecords}
                onDone={handleReset}
                onRetry={handleRetry}
            />
        );
    }

    // Idle: show import selector or child form
    if (!pageState.step) {
        return (
            <BulkInviteSelector
                onSelect={(s) => dispatch({ type: "SELECT_STEP", step: s })}
                title={ROLE_LABELS[role] ?? "Invite Staff Members"}
            />
        );
    }

    if (pageState.step === "manual") {
        return (
            <BulkInviteManualForm
                role={role}
                onReset={handleReset}
                onJobCreated={handleJobCreated}
                submitFn={submitFn}
            />
        );
    }

    if (pageState.step === "file") {
        return (
            <BulkInviteFileImporter
                role={role}
                onReset={handleReset}
                onJobCreated={handleJobCreated}
                submitFn={submitFn}
            />
        );
    }

    return <section />;
}
