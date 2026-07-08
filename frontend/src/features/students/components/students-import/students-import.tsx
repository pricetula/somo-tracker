"use client";

import * as React from "react";
import { Loader2 } from "lucide-react";
import { StudentsImportSelector } from "./import-selector";
import { StudentManualImportForm } from "./manual-import-form";
import { FileImporter } from "./file-importer";
import { ImportProgress } from "./import-progress";
import { getActiveImportJob } from "@/lib/api/imports";
import { useMe } from "@/hooks/use-auth";

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

interface StudentsImportFormProps {
    isDialogVersion: boolean;
}

// ─── Component ────────────────────────────────────────────────────────────

export function StudentsImportForm({ isDialogVersion }: StudentsImportFormProps) {
    const { data: me } = useMe();
    const [pageState, dispatch] = React.useReducer(pageReducer, { phase: "idle", step: null });

    const meResolved = me !== undefined;
    const schoolId = me?.school_id;

    // Fire active-job check once when me resolves with a school_id.
    // The effect depends on meResolved so it only runs after the query settles.
    // The cleanup flag prevents stale dispatches if unmounted mid-check.
    React.useEffect(() => {
        if (!meResolved || !schoolId) return;

        let cancelled = false;

        getActiveImportJob(schoolId)
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
                // Transient — fall through to normal flow (idle with no active job)
            });

        return () => {
            cancelled = true;
        };
    }, [meResolved, schoolId]);

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

    // Loading state is derived purely from the useMe query — no local state needed.
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
            <StudentsImportSelector
                onSelect={(s) => dispatch({ type: "SELECT_STEP", step: s })}
                isDialogVersion={isDialogVersion}
            />
        );
    }

    if (pageState.step === "manual") {
        return <StudentManualImportForm onReset={handleReset} onJobCreated={handleJobCreated} />;
    }

    if (pageState.step === "file") {
        return <FileImporter onReset={handleReset} onJobCreated={handleJobCreated} />;
    }

    return <section />;
}
