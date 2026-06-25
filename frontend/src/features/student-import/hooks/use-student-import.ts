/**
 * Student Import wizard state management.
 *
 * Orchestrates the full state machine: ingestion vector selection → data entry →
 * validation → submission → results.
 *
 * All persistent state is mirrored to IndexedDB for session recovery.
 */

"use client";

import * as React from "react";
import { toast } from "sonner";
import { getErrorMessage } from "@/lib/errors";
import type {
    ImportStep,
    StagedStudentRecord,
    ImportSession,
    MappingConfig,
    ParsedFileResult,
    ImportResultSummary,
    ExistingStudent,
    ParentsMap,
    ClassesMap,
    StudentImportPayload,
} from "../types";

// ─── Academic term helpers ─────────────────────────────────────────────────

export type AcademicTerm = "Term 1" | "Term 2" | "Term 3";

export const VALID_ACADEMIC_YEARS = ["2024", "2025", "2026", "2027"];

export function getCurrentAcademicYear(): string {
    const now = new Date();
    const year = now.getFullYear();
    // Kenyan academic year typically runs Jan-Dec; if we're past June,
    // the current academic year is the calendar year.
    return String(year);
}

export function getCurrentTerm(): AcademicTerm {
    const month = new Date().getMonth() + 1; // 1-indexed
    if (month >= 1 && month <= 4) return "Term 1";
    if (month >= 5 && month <= 8) return "Term 2";
    return "Term 3";
}
import {
    saveSession,
    loadSession,
    clearSession,
    saveRecords,
    loadRecords,
    updateRecord,
} from "../lib/indexeddb";
import { validateRecord, detectDuplicates } from "../lib/validation";
import { submitBulkImport } from "../services/import-api";

// ─── State ─────────────────────────────────────────────────────────────────

export interface StudentImportState {
    step: ImportStep;
    ingestionPattern: "manual" | "csv" | null;

    // Pattern A: manual entry rows (raw form)
    manualRows: ManualRow[];

    // Pattern B: parsed file data
    parsedFile: ParsedFileResult | null;

    // Wizard mapping config
    mappingConfig: MappingConfig;

    // Phase 3: staged validated records
    stagedRecords: StagedStudentRecord[];

    // Phase 4 validation display
    viewFilter: "errors" | "all";

    // Submission
    submitting: boolean;
    resultSummary: ImportResultSummary | null;

    // Session
    sessionId: string;
}

export interface ManualRow {
    _rowIndex: number;
    full_name: string;
    gender: string;
    date_of_birth: string;
    upi_number: string;
    knec_assessment_number: string;
    parent_name: string;
    class_name: string;
}

const DEFAULT_MAPPING: MappingConfig = {
    nameColumns: [],
    genderColumn: null,
    dobColumn: null,
    upiColumn: null,
    knecColumn: null,
    parentColumns: [],
    classColumns: [],
};

function generateSessionId(): string {
    return crypto.randomUUID();
}

// ─── Hook ──────────────────────────────────────────────────────────────────

export function useStudentImport(
    parentsMap: ParentsMap,
    classesMap: ClassesMap,
    existingStudents: ExistingStudent[]
) {
    const [step, setStep] = React.useState<ImportStep>("term-select");
    const [ingestionPattern, setIngestionPattern] = React.useState<"manual" | "csv" | null>(null);
    const [manualRows, setManualRows] = React.useState<ManualRow[]>([
        {
            _rowIndex: 0,
            full_name: "",
            gender: "",
            date_of_birth: "",
            upi_number: "",
            knec_assessment_number: "",
            parent_name: "",
            class_name: "",
        },
    ]);
    const [parsedFile, setParsedFile] = React.useState<ParsedFileResult | null>(null);
    const [mappingConfig, setMappingConfig] = React.useState<MappingConfig>(DEFAULT_MAPPING);
    const [stagedRecords, setStagedRecords] = React.useState<StagedStudentRecord[]>([]);
    const [viewFilter, setViewFilter] = React.useState<"errors" | "all">("errors");
    const [submitting, setSubmitting] = React.useState(false);
    const [resultSummary, setResultSummary] = React.useState<ImportResultSummary | null>(null);
    const [academicYear, setAcademicYear] = React.useState<string>(getCurrentAcademicYear());
    const [term, setTerm] = React.useState<string>(getCurrentTerm());
    const [sessionId] = React.useState(generateSessionId);

    // ── Session Persistence ──────────────────────────────────────────────

    const persistSession = React.useCallback(
        async (currentStep: ImportStep) => {
            const session: ImportSession = {
                sessionId,
                createdAt: new Date().toISOString(),
                lastUpdatedAt: new Date().toISOString(),
                currentStep,
                totalRecords: stagedRecords.length || manualRows.length,
                ingestionPattern: ingestionPattern ?? "manual",
                mappingConfig,
                academicYear: academicYear,
                term,
            };
            await saveSession(session);
        },
        [
            sessionId,
            stagedRecords.length,
            manualRows.length,
            ingestionPattern,
            mappingConfig,
            academicYear,
            term,
        ]
    );

    // Persist on step changes
    React.useEffect(() => {
        if (step !== "selector" && step !== "staging") {
            persistSession(step);
        }
    }, [step, persistSession]);

    // ── Session Recovery ─────────────────────────────────────────────────

    const restoreFromSession = React.useCallback(async () => {
        const session = await loadSession();
        if (!session) return;

        setStep(session.currentStep);
        setIngestionPattern(session.ingestionPattern);
        setMappingConfig(session.mappingConfig);
        setAcademicYear(session.academicYear ?? getCurrentAcademicYear());
        setTerm(session.term ?? getCurrentTerm());

        const records = await loadRecords();
        if (records.length > 0) {
            setStagedRecords(records);
        }

        toast.info("Session restored", {
            description: `Resumed import from ${new Date(session.createdAt).toLocaleString()}`,
        });
    }, []);

    // ── Pattern A: Manual Row Management ─────────────────────────────────

    const addManualRow = React.useCallback(() => {
        setManualRows((prev) => [
            ...prev,
            {
                _rowIndex: prev.length,
                full_name: "",
                gender: "",
                date_of_birth: "",
                upi_number: "",
                knec_assessment_number: "",
                parent_name: "",
                class_name: "",
            },
        ]);
    }, []);

    const removeManualRow = React.useCallback((rowIndex: number) => {
        setManualRows((prev) => prev.filter((r) => r._rowIndex !== rowIndex));
    }, []);

    const updateManualRow = React.useCallback(
        (rowIndex: number, field: keyof ManualRow, value: string) => {
            setManualRows((prev) =>
                prev.map((r) => (r._rowIndex !== rowIndex ? r : { ...r, [field]: value }))
            );
        },
        []
    );

    // ── Pattern B: File Parsing ──────────────────────────────────────────

    const setParsedFileData = React.useCallback((file: ParsedFileResult) => {
        setParsedFile(file);
    }, []);

    // ── Mapping Config ───────────────────────────────────────────────────

    const updateMapping = React.useCallback((update: Partial<MappingConfig>) => {
        setMappingConfig((prev) => ({ ...prev, ...update }));
    }, []);

    // ── Phase 3: Stage Records ───────────────────────────────────────────

    const stageRecords = React.useCallback(async () => {
        setStep("staging");

        let rawRows: Record<string, string>[] = [];

        if (ingestionPattern === "manual") {
            rawRows = manualRows.map((row) => ({
                full_name: row.full_name,
                gender: row.gender,
                date_of_birth: row.date_of_birth,
                upi_number: row.upi_number,
                knec_assessment_number: row.knec_assessment_number,
                parent_name: row.parent_name,
                class_name: row.class_name,
            }));
        } else if (ingestionPattern === "csv" && parsedFile) {
            rawRows = parsedFile.fullData;
        }

        // Run validation on each row
        // For manual mode, construct a mapping that maps fields directly
        let effectiveMapping: MappingConfig;

        if (ingestionPattern === "manual") {
            effectiveMapping = {
                nameColumns: ["full_name"],
                genderColumn: "gender",
                dobColumn: "date_of_birth",
                upiColumn: "upi_number",
                knecColumn: "knec_assessment_number",
                parentColumns: ["parent_name"],
                classColumns: ["class_name"],
            };
        } else {
            effectiveMapping = mappingConfig;
        }

        const parentsLookup = parentsMap.size > 0 ? parentsMap : null;
        const classesLookup = classesMap.size > 0 ? classesMap : null;

        const validated = rawRows.map((raw, index) =>
            validateRecord(index, raw, effectiveMapping, parentsLookup, classesLookup)
        );

        // Run duplicate detection
        const withDuplicates = detectDuplicates(validated, existingStudents);

        await saveRecords(withDuplicates);
        setStagedRecords(withDuplicates);
        setStep("validation");
        setViewFilter("errors");
    }, [
        ingestionPattern,
        manualRows,
        parsedFile,
        mappingConfig,
        parentsMap,
        classesMap,
        existingStudents,
    ]);

    // ── Phase 4: Record Updates ──────────────────────────────────────────

    const updateStagedRecord = React.useCallback(
        async (rowIndex: number, update: Partial<StagedStudentRecord>) => {
            const updated = await updateRecord(rowIndex, update);
            setStagedRecords(updated);
        },
        []
    );

    const toggleImportAnyway = React.useCallback(async (rowIndex: number) => {
        const records = await loadRecords();
        const record = records.find((r) => r._rowIndex === rowIndex);
        if (!record) return;
        const updated = await updateRecord(rowIndex, {
            importAnyway: !record.importAnyway,
        });
        setStagedRecords(updated);
    }, []);

    // ── Filter / count helpers ───────────────────────────────────────────

    const errorAndWarningCount = React.useMemo(() => {
        const filtered = stagedRecords.filter(
            (r) => !r.isValid || (r.isDuplicate && !r.importAnyway)
        );
        return {
            errorCount: filtered.filter((r) => !r.isValid).length,
            duplicateWarningCount: filtered.filter(
                (r) => r.isDuplicate && !r.importAnyway && r.isValid
            ).length,
            totalErrorCount: filtered.length,
        };
    }, [stagedRecords]);

    const visibleRecords = React.useMemo(() => {
        if (viewFilter === "all") return stagedRecords;
        return stagedRecords.filter((r) => !r.isValid || (r.isDuplicate && !r.importAnyway));
    }, [stagedRecords, viewFilter]);

    const resolvedRecords = React.useMemo(
        () => stagedRecords.filter((r) => r.isValid && !r.isDuplicate),
        [stagedRecords]
    );

    const clearResolvedRows = React.useCallback(() => {
        // Keep resolved rows visible per spec — this just marks them
        // The "Clear Resolved Rows" removes them from the visible list
        // but keeps them in IndexedDB for submission
        setViewFilter("all");
    }, []);

    // ── Submit ───────────────────────────────────────────────────────────

    const submitImport = React.useCallback(async () => {
        setSubmitting(true);
        setStep("submitting");

        try {
            // Build payload from all valid + overridden records
            const payload: StudentImportPayload[] = stagedRecords
                .filter((r) => r.isValid || r.importAnyway)
                .filter((r) => !r.isDuplicate || r.importAnyway)
                .map((r) => ({
                    full_name: r.full_name,
                    gender: r.gender as "M" | "F",
                    date_of_birth: r.date_of_birth ?? undefined,
                    upi_number: r.upi_number ?? undefined,
                    knec_assessment_number: r.knec_assessment_number ?? undefined,
                    cbc_student_parents_id: r.cbc_student_parents_id ?? undefined,
                    class_id: r.class_id ?? undefined,
                }));

            if (payload.length === 0) {
                toast.error("No valid records to submit");
                setSubmitting(false);
                return;
            }

            const result = await submitBulkImport(academicYear, term, payload);

            setResultSummary(result.summary);
            setStep("results");

            if (result.summary.status === "success") {
                toast.success(`Successfully imported ${result.summary.successCount} students`);
                // Purge session on full success
                await clearSession();
            } else if (result.summary.status === "partial") {
                toast.warning(
                    `Imported ${result.summary.successCount} students. ${result.summary.failureCount} failed.`
                );
            } else {
                toast.error(result.summary.message ?? "Import failed");
            }
        } catch (err: unknown) {
            toast.error(getErrorMessage(err));
            setStep("validation");
        } finally {
            setSubmitting(false);
        }
    }, [stagedRecords, academicYear, term]);

    // ── Reset ────────────────────────────────────────────────────────────

    const resetImport = React.useCallback(async () => {
        await clearSession();
        setStep("term-select");
        setIngestionPattern(null);
        setAcademicYear(getCurrentAcademicYear());
        setTerm(getCurrentTerm());
        setManualRows([
            {
                _rowIndex: 0,
                full_name: "",
                gender: "",
                date_of_birth: "",
                upi_number: "",
                knec_assessment_number: "",
                parent_name: "",
                class_name: "",
            },
        ]);
        setParsedFile(null);
        setMappingConfig(DEFAULT_MAPPING);
        setStagedRecords([]);
        setViewFilter("errors");
        setSubmitting(false);
        setResultSummary(null);
    }, []);

    return {
        // State
        step,
        ingestionPattern,
        manualRows,
        parsedFile,
        mappingConfig,
        stagedRecords,
        viewFilter,
        submitting,
        resultSummary,
        errorAndWarningCount,
        visibleRecords,
        resolvedRecords,
        academicYear,
        term,

        // Actions
        setStep,
        setIngestionPattern,
        setAcademicYear,
        setTerm,
        addManualRow,
        removeManualRow,
        updateManualRow,
        setParsedFileData,
        updateMapping,
        stageRecords,
        updateStagedRecord,
        toggleImportAnyway,
        setViewFilter,
        clearResolvedRows,
        submitImport,
        resetImport,
        restoreFromSession,
    };
}
