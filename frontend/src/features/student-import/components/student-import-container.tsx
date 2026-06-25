/**
 * Student Import Container — the main entry point for the student bulk import feature.
 *
 * Handles session recovery on mount, orchestrates PhD 1-4 phases,
 * and manages the full wizard flow.
 */

"use client";

import * as React from "react";
import { SessionRecoveryBanner } from "./session-recovery-banner";
import { LookupWarningBanner } from "./lookup-warning-banner";
import { IngestionSelector } from "./ingestion-selector";
import { ManualEntryGrid } from "./manual-entry-grid";
import { FileDropzone } from "./file-dropzone";
import { ColumnWizard } from "./column-wizard";
import { ValidationMotor } from "./validation-motor";
import { ResultsSummary } from "./results-summary";
import { TermSelectionStep } from "./term-selection-step";
import { useSessionRecovery } from "../hooks/use-session-recovery";
import { useLookups } from "../hooks/use-lookups";
import { useStudentImport } from "../hooks/use-student-import";
import type { ParsedFileResult } from "../types";

export function StudentImportContainer() {
    // ── Session Recovery ──────────────────────────────────────────────────
    const { action, session, resume, discard } = useSessionRecovery();

    // ── Lookups (Phase 1) ─────────────────────────────────────────────────
    const {
        parentsMap,
        classesMap,
        existingStudents,
        parentsError,
        classesError,
        parentsLoading,
        classesLoading,
        retryParents,
        retryClasses,
    } = useLookups();

    // ── Wizard State (Phases 2-4) ─────────────────────────────────────────
    const wizard = useStudentImport(parentsMap, classesMap, existingStudents);

    // ── Session Recovery Handler ──────────────────────────────────────────

    React.useEffect(() => {
        if (action === "prompt" && session) {
            // Don't auto-resume — wait for user action via banner
        }
        if (action === "clear") {
            // No session — wizard is at default selector step
        }
    }, [action, session]);

    function handleResume() {
        resume();
        wizard.restoreFromSession();
    }

    function handleDiscard() {
        discard();
        wizard.resetImport();
    }

    // ── Term Selected → Show Ingestion Selector ──────────────────────────

    function handleTermContinue() {
        wizard.setStep("selector");
    }

    // ── Pattern A: Manual Entry → Stage ───────────────────────────────────

    function handleManualProceed() {
        wizard.stageRecords();
    }

    // ── Pattern B: File Upload → Wizard → Stage ──────────────────────────

    function handleFileParsed(file: ParsedFileResult) {
        wizard.setIngestionPattern("csv");
        wizard.setParsedFileData(file);
        // Wizard steps will follow
    }

    function handleWizardComplete() {
        wizard.stageRecords();
    }

    // ── Submission Retry ──────────────────────────────────────────────────

    function handleRetry() {
        wizard.submitImport();
    }

    // ── Start New ─────────────────────────────────────────────────────────

    function handleStartNew() {
        wizard.resetImport();
    }

    // ── Lookup loading / degraded rendering ───────────────────────────────

    const isLookupsLoading = parentsLoading || classesLoading;

    // ── Render ────────────────────────────────────────────────────────────

    return (
        <div className="flex flex-col">
            {/* Session recovery banner */}
            {action === "loading" && (
                <div className="bg-muted/30 px-4 py-2">
                    <p className="text-muted-foreground text-xs">Checking for existing sessions…</p>
                </div>
            )}

            {action === "prompt" && session && (
                <SessionRecoveryBanner
                    session={session}
                    onResume={handleResume}
                    onDiscard={handleDiscard}
                />
            )}

            {/* Lookup warning banners */}
            {parentsError && (
                <LookupWarningBanner type="parents" message={parentsError} onRetry={retryParents} />
            )}
            {classesError && (
                <LookupWarningBanner type="classes" message={classesError} onRetry={retryClasses} />
            )}

            {/* Lookup degraded inline notes */}
            {parentsError && wizard.step === "validation" && (
                <div className="bg-muted/20 text-muted-foreground px-3 py-1.5 text-xs">
                    Parent linking unavailable — lookup failed.
                </div>
            )}
            {classesError && wizard.step === "validation" && (
                <div className="bg-muted/20 text-muted-foreground px-3 py-1.5 text-xs">
                    Class linking unavailable — lookup failed.
                </div>
            )}

            {/* Loading state for lookups */}
            {isLookupsLoading && (wizard.step === "term-select" || wizard.step === "selector") && (
                <div className="bg-muted/20 text-muted-foreground px-3 py-1.5 text-xs">
                    Loading reference data…
                </div>
            )}

            {/* Main content */}
            <div className="flex-1 px-4 py-4">
                {wizard.step === "term-select" && (
                    <TermSelectionStep
                        academicYear={wizard.academicYear}
                        term={wizard.term}
                        onAcademicYearChange={wizard.setAcademicYear}
                        onTermChange={wizard.setTerm}
                        onContinue={handleTermContinue}
                    />
                )}

                {wizard.step === "selector" && !isLookupsLoading && (
                    <IngestionSelector
                        onSelect={(mode) => {
                            if (mode === "manual") {
                                wizard.setIngestionPattern("manual");
                                wizard.setStep("manual-entry");
                            } else {
                                wizard.setIngestionPattern("csv");
                                wizard.setStep("file-wizard");
                            }
                        }}
                    />
                )}

                {wizard.step === "manual-entry" && (
                    <ManualEntryGrid
                        rows={wizard.manualRows}
                        parentsMap={parentsMap}
                        classesMap={classesMap}
                        onAddRow={wizard.addManualRow}
                        onRemoveRow={wizard.removeManualRow}
                        onUpdateRow={wizard.updateManualRow}
                        onProceed={handleManualProceed}
                    />
                )}

                {wizard.step === "file-wizard" && !wizard.parsedFile && (
                    <FileDropzone
                        onFileParsed={handleFileParsed}
                        onBack={() => wizard.setStep("selector")}
                    />
                )}

                {wizard.step === "file-wizard" && wizard.parsedFile && (
                    <ColumnWizard
                        parsedFile={wizard.parsedFile}
                        mappingConfig={wizard.mappingConfig}
                        onMappingChange={wizard.updateMapping}
                        onComplete={handleWizardComplete}
                    />
                )}

                {wizard.step === "staging" && (
                    <div className="flex items-center justify-center py-16">
                        <p className="text-muted-foreground text-sm">Processing records…</p>
                    </div>
                )}

                {wizard.step === "validation" && (
                    <ValidationMotor
                        records={wizard.visibleRecords}
                        viewFilter={wizard.viewFilter}
                        onViewFilterChange={wizard.setViewFilter}
                        onUpdateRecord={wizard.updateStagedRecord}
                        onToggleImportAnyway={wizard.toggleImportAnyway}
                        errorCount={wizard.errorAndWarningCount.errorCount}
                        duplicateWarningCount={wizard.errorAndWarningCount.duplicateWarningCount}
                        totalErrorCount={wizard.errorAndWarningCount.totalErrorCount}
                        submitting={wizard.submitting}
                        onSubmit={wizard.submitImport}
                        onBack={() => wizard.setStep("selector")}
                    />
                )}

                {wizard.step === "submitting" && (
                    <div className="flex items-center justify-center py-16">
                        <p className="text-muted-foreground text-sm">Submitting import…</p>
                    </div>
                )}

                {wizard.step === "results" && wizard.resultSummary && (
                    <ResultsSummary
                        summary={wizard.resultSummary}
                        onRetry={handleRetry}
                        onStartNew={handleStartNew}
                    />
                )}
            </div>
        </div>
    );
}
