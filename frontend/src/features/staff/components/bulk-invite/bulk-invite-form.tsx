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
import { ImportProgress } from "@/features/students/components/students-import/import-progress";
import { getActiveImportJob } from "@/lib/api/imports";
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

interface BulkInviteFormProps {
    /** The role to pre-select. Determines who gets invited. */
    role: "SCHOOL_ADMIN" | "TEACHER" | "NURSE" | "FINANCE";
    isDialogVersion?: boolean;
}

// ─── Component ────────────────────────────────────────────────────────────

export function BulkInviteForm({ role, isDialogVersion = false }: BulkInviteFormProps) {
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
            .catch(() => {
                // Transient — fall through to normal flow
            });

        return () => {
            cancelled = true;
        };
    }, [meResolved]);

    // ── Handlers ──────────────────────────────────────────────────────

    function handleReset() {
        dispatch({ type: "RESET" });
    }

    function handleJobCreated(jobId: string, totalRecords: number) {
        dispatch({ type: "JOB_CREATED", job: { jobId, totalRecords } });
    }

    function handleRetry() {
        dispatch({ type: "RESET" });
    }

    // ── Render ────────────────────────────────────────────────────────

    if (!meResolved) {
        return (
            <div className="flex items-center justify-center py-12">
                <Loader2 className="text-muted-foreground size-5 animate-spin" />
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
                isDialogVersion={isDialogVersion}
            />
        );
    }

    if (pageState.step === "manual") {
        return (
            <BulkInviteManualForm
                role={role}
                onReset={handleReset}
                onJobCreated={handleJobCreated}
            />
        );
    }

    if (pageState.step === "file") {
        return (
            <BulkInviteFileImporter
                role={role}
                onReset={handleReset}
                onJobCreated={handleJobCreated}
            />
        );
    }

    return <section />;
}
