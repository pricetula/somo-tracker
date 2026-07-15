/**
 * StudentManualImportForm — bulk-add students via manual entry.
 *
 * Validates:
 * - Required fields (full_name)
 * - Within-batch duplicates (admission_number, upi_number, knec_assessment_number)
 * - Against-existing DB records (via POST /api/v1/students/check-duplicates on submit)
 *
 * POSTs all rows to POST /api/v1/students/import (single request),
 * then delegates to the shared <ImportProgress /> for polling + results.
 */

"use client";

import * as React from "react";
import { Plus, Trash2, Loader2 } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { Label } from "@/components/ui/label";
import { DatePicker } from "@/components/ui/date-picker";
import { ClassCombobox } from "@/features/classes";

import {
    submitStudentImport,
    checkDuplicates,
    getImportAlreadyInProgress,
} from "@/lib/api/imports";
import type { ImportRow } from "@/lib/api/imports";
import { getErrorMessage } from "@/lib/errors";

// ─── Constants ────────────────────────────────────────────────────────────

// MAX_IMPORT_ROWS must stay in sync with backend imports.MaxImportRows (5000).
// This is a proactive UX guard — the backend is the source of truth.
const MAX_IMPORT_ROWS = 5000;

// ─── Types ────────────────────────────────────────────────────────────────

interface StudentRow {
    id: string;
    fullName: string;
    gender: string;
    dateOfBirth: string;
    upiNumber: string;
    knecNumber: string;
    admissionNumber: string;
    classId: string;
}

let rowCounter = 0;

function freshRow(): StudentRow {
    rowCounter += 1;
    return {
        id: `row-${rowCounter}-${Date.now()}`,
        fullName: "",
        gender: "",
        dateOfBirth: "",
        upiNumber: "",
        knecNumber: "",
        admissionNumber: "",
        classId: "",
    };
}

// ─── Within-batch duplicate detection ─────────────────────────────────────

interface BatchDupResult {
    /** rowId -> { fieldName -> error message } */
    errors: Record<string, Record<string, string>>;
    hasErrors: boolean;
}

/**
 * Detect within-batch duplicates among the current set of rows.
 * Does NOT call the API — purely client-side.
 */
function detectWithinBatchDuplicates(rows: StudentRow[]): BatchDupResult {
    const errors: Record<string, Record<string, string>> = {};
    let hasErrors = false;

    // Build value-to-row mappings for each tracked field (case-insensitive)
    const admMap = new Map<string, string[]>(); // value -> rowIds
    const upiMap = new Map<string, string[]>();
    const knecMap = new Map<string, string[]>();

    for (const row of rows) {
        if (row.admissionNumber.trim()) {
            const key = row.admissionNumber.trim().toLowerCase();
            const ids = admMap.get(key) ?? [];
            ids.push(row.id);
            admMap.set(key, ids);
        }
        if (row.upiNumber.trim()) {
            const key = row.upiNumber.trim().toLowerCase();
            const ids = upiMap.get(key) ?? [];
            ids.push(row.id);
            upiMap.set(key, ids);
        }
        if (row.knecNumber.trim()) {
            const key = row.knecNumber.trim().toLowerCase();
            const ids = knecMap.get(key) ?? [];
            ids.push(row.id);
            knecMap.set(key, ids);
        }
    }

    // Helper: for each duplicate map, assign error messages
    function markFieldDups(map: Map<string, string[]>, fieldName: string, label: string) {
        for (const [, rowIds] of map) {
            if (rowIds.length <= 1) continue;
            hasErrors = true;
            for (const rowId of rowIds) {
                const otherRowNumbers = rows
                    .filter((r) => r.id !== rowId && rowIds.includes(r.id))
                    .map((r) => rows.findIndex((x) => x.id === r.id) + 1)
                    .filter((n) => n > 0);
                if (otherRowNumbers.length === 0) continue;
                if (!errors[rowId]) errors[rowId] = {};
                const existing = errors[rowId][fieldName] ?? "";
                const msg = `Duplicate ${label} — also used in row${otherRowNumbers.length > 1 ? "s" : ""} ${otherRowNumbers.join(", ")}`;
                errors[rowId][fieldName] = existing ? `${existing}; ${msg}` : msg;
            }
        }
    }

    markFieldDups(admMap, "admissionNumber", "admission number");
    markFieldDups(upiMap, "upiNumber", "UPI number");
    markFieldDups(knecMap, "knecNumber", "KNEC number");

    return { errors, hasErrors };
}

// ─── Props ────────────────────────────────────────────────────────────────

interface StudentManualImportFormProps {
    onReset: () => void;
    onJobCreated: (jobId: string, totalRecords: number) => void;
}

// ─── Component ────────────────────────────────────────────────────────────

export function StudentManualImportForm({ onReset, onJobCreated }: StudentManualImportFormProps) {
    const [rows, setRows] = React.useState<StudentRow[]>([freshRow()]);
    const [submitting, setSubmitting] = React.useState(false);
    const [checkingDuplicates, setCheckingDuplicates] = React.useState(false);
    const submittingRef = React.useRef(false);
    // rowId -> { fieldName -> error message }
    const [fieldErrors, setFieldErrors] = React.useState<Record<string, Record<string, string>>>(
        {}
    );
    // Idempotency key: generated at submit start, persists across retries
    const idempotencyKeyRef = React.useRef<string | null>(null);

    // ── Within-batch duplicate detection (on every render) ─────────────
    const batchDupResult = React.useMemo(() => detectWithinBatchDuplicates(rows), [rows]);
    const hasWithinBatchDups = batchDupResult.hasErrors;

    // Merge within-batch errors into fieldErrors for display.
    // API (existing-record) errors are set separately during submit attempt.
    const mergedFieldErrors = React.useMemo(() => {
        const merged: Record<string, Record<string, string>> = {};
        // Start with within-batch dups
        for (const [rowId, fields] of Object.entries(batchDupResult.errors)) {
            merged[rowId] = { ...fields };
        }
        // Overlay any API/existing errors
        for (const [rowId, fields] of Object.entries(fieldErrors)) {
            if (!merged[rowId]) merged[rowId] = {};
            Object.assign(merged[rowId], fields);
        }
        return merged;
    }, [batchDupResult, fieldErrors]);

    // ── Row mutation helpers ───────────────────────────────────────────
    const updateRow = React.useCallback((rowId: string, patch: Partial<StudentRow>) => {
        setRows((prev) => prev.map((r) => (r.id === rowId ? { ...r, ...patch } : r)));
        // Clear existing-record errors for this field when user edits it
        setFieldErrors((prev) => {
            const rowErrors = prev[rowId];
            if (!rowErrors) return prev;
            const updated = { ...rowErrors };
            for (const key of Object.keys(patch)) {
                // Map camelCase JS property names to snake_case error keys
                delete updated[key];
                if (key === "fullName") {
                    delete updated.full_name;
                }
            }
            if (Object.keys(updated).length === 0) {
                const { [rowId]: _removed, ...rest } = prev;
                return rest;
            }
            return { ...prev, [rowId]: updated };
        });
    }, []);

    const removeRow = React.useCallback((rowId: string) => {
        setRows((prev) => prev.filter((r) => r.id !== rowId));
    }, []);

    const atMaxRows = rows.length >= MAX_IMPORT_ROWS;

    const addRow = React.useCallback(() => {
        setRows((prev) => {
            if (prev.length >= MAX_IMPORT_ROWS) {
                toast.error(
                    `Maximum of ${MAX_IMPORT_ROWS.toLocaleString()} rows reached. Please split into smaller imports.`
                );
                return prev;
            }
            return [freshRow(), ...prev];
        });
    }, []);

    // ── Against-existing-records check ─────────────────────────────────
    const checkExistingDuplicates = React.useCallback(async (): Promise<boolean> => {
        const nonEmptyRows = rows.filter(
            (r) => r.admissionNumber.trim() || r.upiNumber.trim() || r.knecNumber.trim()
        );
        if (nonEmptyRows.length === 0) return true; // nothing to check

        const admNumbers = rows.map((r) => r.admissionNumber.trim()).filter(Boolean);
        const upiNumbers = rows.map((r) => r.upiNumber.trim()).filter(Boolean);
        const knecNumbers = rows.map((r) => r.knecNumber.trim()).filter(Boolean);

        // If all are empty, skip
        if (admNumbers.length === 0 && upiNumbers.length === 0 && knecNumbers.length === 0) {
            return true;
        }

        setCheckingDuplicates(true);
        try {
            const result = await checkDuplicates({
                admission_numbers: admNumbers,
                upi_numbers: upiNumbers,
                knec_assessment_numbers: knecNumbers,
            });

            const existingAdmSet = new Set(
                result.existing_admission_numbers.map((v) => v.toLowerCase())
            );
            const existingUPISet = new Set(result.existing_upi_numbers.map((v) => v.toLowerCase()));
            const existingKnecSet = new Set(
                result.existing_knec_assessment_numbers.map((v) => v.toLowerCase())
            );

            const newErrors: Record<string, Record<string, string>> = {};
            let hasConflicts = false;

            for (const row of rows) {
                if (
                    row.admissionNumber.trim() &&
                    existingAdmSet.has(row.admissionNumber.trim().toLowerCase())
                ) {
                    if (!newErrors[row.id]) newErrors[row.id] = {};
                    newErrors[row.id].admissionNumber =
                        `Admission number "${row.admissionNumber.trim()}" already exists for this school`;
                    hasConflicts = true;
                }
                if (
                    row.upiNumber.trim() &&
                    existingUPISet.has(row.upiNumber.trim().toLowerCase())
                ) {
                    if (!newErrors[row.id]) newErrors[row.id] = {};
                    newErrors[row.id].upiNumber =
                        `UPI number "${row.upiNumber.trim()}" already exists for this school`;
                    hasConflicts = true;
                }
                if (
                    row.knecNumber.trim() &&
                    existingKnecSet.has(row.knecNumber.trim().toLowerCase())
                ) {
                    if (!newErrors[row.id]) newErrors[row.id] = {};
                    newErrors[row.id].knecNumber =
                        `KNEC number "${row.knecNumber.trim()}" already exists for this school`;
                    hasConflicts = true;
                }
            }

            setFieldErrors(newErrors);
            return !hasConflicts;
        } catch (err) {
            console.error("Failed to check duplicates:", err);
            // If the check fails, allow submission to proceed (safety net handles it)
            return true;
        } finally {
            setCheckingDuplicates(false);
        }
    }, [rows]);

    // ── Validation + Submit ────────────────────────────────────────────
    const handleImport = React.useCallback(async () => {
        if (submittingRef.current) return;

        // Validate required fields
        const requiredErrors: Record<string, Record<string, string>> = {};
        let hasRequiredErrors = false;
        rows.forEach((row) => {
            if (!row.fullName.trim()) {
                requiredErrors[row.id] = { full_name: "Full name is required" };
                hasRequiredErrors = true;
            }
        });
        setFieldErrors((prev) => ({ ...prev, ...requiredErrors }));

        // Block if required fields missing or within-batch duplicates
        if (hasRequiredErrors) return;
        if (hasWithinBatchDups) {
            toast.error("Please resolve duplicate values within the batch before submitting");
            return;
        }

        // Check against existing records before submitting
        const canProceed = await checkExistingDuplicates();
        if (!canProceed) {
            toast.error("Some values already exist for this school. Please correct them.");
            return;
        }

        submittingRef.current = true;
        setSubmitting(true);

        // Generate idempotency key on first submit attempt; reuse on retry
        if (!idempotencyKeyRef.current) {
            idempotencyKeyRef.current = crypto.randomUUID();
        }

        const importRows: ImportRow[] = rows.map((r) => ({
            full_name: r.fullName.trim(),
            gender: (r.gender || "M") as "M" | "F",
            date_of_birth: r.dateOfBirth || null,
            upi_number: r.upiNumber || null,
            knec_assessment_number: r.knecNumber || null,
            admission_number: r.admissionNumber || null,
            class_id: r.classId || undefined,
        }));

        try {
            const result = await submitStudentImport({
                idempotency_key: idempotencyKeyRef.current,
                rows: importRows,
            });
            // Key consumed on success
            idempotencyKeyRef.current = null;
            onJobCreated(result.job_id, importRows.length);
        } catch (err) {
            // Handle import_already_in_progress: redirect to existing job's progress
            const activeJobId = getImportAlreadyInProgress(err);
            if (activeJobId) {
                onJobCreated(activeJobId, importRows.length);
                return;
            }

            // Keep the same key for retry (transient network failure)
            submittingRef.current = false;
            setSubmitting(false);
            toast.error(getErrorMessage(err));
        }
    }, [rows, onJobCreated, hasWithinBatchDups, checkExistingDuplicates]);

    const isSubmitting = submitting || checkingDuplicates;

    // ── Determine blocking errors count ────────────────────────────────
    const blockingCount = React.useMemo(() => {
        let count = 0;
        for (const row of rows) {
            const errors = mergedFieldErrors[row.id];
            if (errors && Object.keys(errors).length > 0) count++;
        }
        return count;
    }, [rows, mergedFieldErrors]);

    // ── Render helpers ─────────────────────────────────────────────────
    function fieldError(rowId: string, field: string): string | undefined {
        return mergedFieldErrors[rowId]?.[field];
    }

    return (
        <div className="flex flex-col gap-4">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div>
                    <h3 className="font-medium">Manual Student Import</h3>
                    {blockingCount > 0 && (
                        <p className="text-destructive mt-0.5 text-xs">
                            {blockingCount} row{blockingCount !== 1 ? "s" : ""} with errors
                        </p>
                    )}
                </div>
                <div className="flex items-center gap-2">
                    <Button
                        variant="ghost"
                        size="sm"
                        onClick={addRow}
                        disabled={isSubmitting || atMaxRows}
                        title={
                            atMaxRows
                                ? `Maximum of ${MAX_IMPORT_ROWS.toLocaleString()} rows reached`
                                : undefined
                        }
                    >
                        <Plus className="mr-1 size-3.5" />
                        Add Row
                    </Button>
                    <Button variant="outline" size="sm" onClick={onReset} disabled={isSubmitting}>
                        Cancel
                    </Button>
                </div>
            </div>

            {/* Rows */}
            {rows.length === 0 ? (
                <div className="text-muted-foreground flex flex-col items-center gap-3 py-12 text-xs">
                    <p>No students added yet.</p>
                    <Button variant="outline" size="sm" onClick={addRow}>
                        <Plus className="mr-1 size-3.5" />
                        Add a student
                    </Button>
                </div>
            ) : (
                <div className="space-y-2">
                    {rows.map((row, index) => {
                        const rowMergedErrors = mergedFieldErrors[row.id] ?? {};
                        const hasRowErrors = Object.keys(rowMergedErrors).length > 0;

                        return (
                            <div
                                key={row.id}
                                className={`rounded-md p-3 ${hasRowErrors ? "bg-destructive/5 ring-destructive/20 ring-1" : "bg-muted/30"}`}
                            >
                                <div className="mb-2 flex items-center justify-between">
                                    <span className="text-muted-foreground text-xs font-medium">
                                        Student {index + 1}
                                        {hasRowErrors && (
                                            <span className="text-destructive ml-1">(errors)</span>
                                        )}
                                    </span>
                                    <Button
                                        variant="ghost"
                                        size="icon-xs"
                                        onClick={() => removeRow(row.id)}
                                        disabled={isSubmitting || rows.length === 1}
                                        className="text-muted-foreground hover:text-destructive size-6"
                                        aria-label={`Remove student ${index + 1}`}
                                    >
                                        <Trash2 className="size-3.5" />
                                    </Button>
                                </div>

                                {/* Full Name + Gender */}
                                <div className="mb-2 grid grid-cols-1 gap-2 sm:grid-cols-2">
                                    <div className="space-y-1">
                                        <Label
                                            className="text-[0.625rem]"
                                            htmlFor={`name-${row.id}`}
                                        >
                                            Full Name <span className="text-destructive">*</span>
                                        </Label>
                                        <Input
                                            id={`name-${row.id}`}
                                            value={row.fullName}
                                            onChange={(e) =>
                                                updateRow(row.id, { fullName: e.target.value })
                                            }
                                            placeholder="e.g. John Kiprop"
                                            disabled={isSubmitting}
                                            className="h-8 text-xs"
                                        />
                                        {fieldError(row.id, "full_name") && (
                                            <p className="text-destructive text-[0.625rem]">
                                                {fieldError(row.id, "full_name")}
                                            </p>
                                        )}
                                    </div>
                                    <div className="space-y-1">
                                        <Label
                                            className="text-[0.625rem]"
                                            htmlFor={`gender-${row.id}`}
                                        >
                                            Gender
                                        </Label>
                                        <Select
                                            value={row.gender}
                                            onValueChange={(v) => updateRow(row.id, { gender: v })}
                                            disabled={isSubmitting}
                                        >
                                            <SelectTrigger
                                                id={`gender-${row.id}`}
                                                className="h-8 text-xs"
                                            >
                                                <SelectValue placeholder="-" />
                                            </SelectTrigger>
                                            <SelectContent>
                                                <SelectItem value="M">Male</SelectItem>
                                                <SelectItem value="F">Female</SelectItem>
                                            </SelectContent>
                                        </Select>
                                    </div>
                                </div>

                                {/* DOB + UPI */}
                                <div className="mb-2 grid grid-cols-1 gap-2 sm:grid-cols-2">
                                    <div className="space-y-1">
                                        <Label
                                            className="text-[0.625rem]"
                                            htmlFor={`dob-${row.id}`}
                                        >
                                            Date of Birth
                                        </Label>
                                        <DatePicker
                                            id={`dob-${row.id}`}
                                            value={row.dateOfBirth}
                                            onChange={(v) => updateRow(row.id, { dateOfBirth: v })}
                                            placeholder="-"
                                            disabled={isSubmitting}
                                        />
                                    </div>
                                    <div className="space-y-1">
                                        <Label
                                            className="text-[0.625rem]"
                                            htmlFor={`upi-${row.id}`}
                                        >
                                            UPI Number
                                        </Label>
                                        <Input
                                            id={`upi-${row.id}`}
                                            value={row.upiNumber}
                                            onChange={(e) =>
                                                updateRow(row.id, { upiNumber: e.target.value })
                                            }
                                            placeholder="e.g. UP123456789"
                                            disabled={isSubmitting}
                                            className={`h-8 text-xs ${fieldError(row.id, "upiNumber") ? "border-destructive" : ""}`}
                                        />
                                        {fieldError(row.id, "upiNumber") && (
                                            <p className="text-destructive text-[0.625rem]">
                                                {fieldError(row.id, "upiNumber")}
                                            </p>
                                        )}
                                    </div>
                                </div>

                                {/* KNEC + Admission */}
                                <div className="mb-2 grid grid-cols-1 gap-2 sm:grid-cols-2">
                                    <div className="space-y-1">
                                        <Label
                                            className="text-[0.625rem]"
                                            htmlFor={`knec-${row.id}`}
                                        >
                                            KNEC Number
                                        </Label>
                                        <Input
                                            id={`knec-${row.id}`}
                                            value={row.knecNumber}
                                            onChange={(e) =>
                                                updateRow(row.id, { knecNumber: e.target.value })
                                            }
                                            placeholder="e.g. KNEC123456"
                                            disabled={isSubmitting}
                                            className={`h-8 text-xs ${fieldError(row.id, "knecNumber") ? "border-destructive" : ""}`}
                                        />
                                        {fieldError(row.id, "knecNumber") && (
                                            <p className="text-destructive text-[0.625rem]">
                                                {fieldError(row.id, "knecNumber")}
                                            </p>
                                        )}
                                    </div>
                                    <div className="space-y-1">
                                        <Label
                                            className="text-[0.625rem]"
                                            htmlFor={`adm-${row.id}`}
                                        >
                                            Admission Number
                                        </Label>
                                        <Input
                                            id={`adm-${row.id}`}
                                            value={row.admissionNumber}
                                            onChange={(e) =>
                                                updateRow(row.id, {
                                                    admissionNumber: e.target.value,
                                                })
                                            }
                                            placeholder="e.g. ADM001"
                                            disabled={isSubmitting}
                                            className={`h-8 text-xs ${fieldError(row.id, "admissionNumber") ? "border-destructive" : ""}`}
                                        />
                                        {fieldError(row.id, "admissionNumber") && (
                                            <p className="text-destructive text-[0.625rem]">
                                                {fieldError(row.id, "admissionNumber")}
                                            </p>
                                        )}
                                    </div>
                                </div>

                                {/* Class selector */}
                                <div className="space-y-1">
                                    <Label className="text-[0.625rem]" htmlFor={`class-${row.id}`}>
                                        Class
                                    </Label>
                                    <ClassCombobox
                                        value={row.classId}
                                        onChange={(v) =>
                                            updateRow(row.id, { classId: v as string })
                                        }
                                        placeholder="None (no enrollment)"
                                        className="h-8"
                                    />
                                    <p className="text-muted-foreground text-[0.625rem]">
                                        Leave blank to create without enrollment
                                    </p>
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
                        {rows.length.toLocaleString()} / {MAX_IMPORT_ROWS.toLocaleString()} rows
                        {blockingCount > 0 && (
                            <span className="text-destructive ml-1">
                                · {blockingCount} with issues
                            </span>
                        )}
                    </div>
                    <div className="flex items-center gap-2">
                        <Button
                            variant="outline"
                            size="sm"
                            onClick={addRow}
                            disabled={isSubmitting || atMaxRows}
                            title={
                                atMaxRows
                                    ? `Maximum of ${MAX_IMPORT_ROWS.toLocaleString()} rows reached`
                                    : undefined
                            }
                        >
                            <Plus className="mr-1 size-3.5" /> Add Row
                        </Button>
                        <Button
                            size="sm"
                            onClick={handleImport}
                            disabled={
                                isSubmitting ||
                                rows.length === 0 ||
                                hasWithinBatchDups ||
                                blockingCount > 0
                            }
                        >
                            {submitting ? (
                                <>
                                    <Loader2 className="mr-1.5 size-3.5 animate-spin" /> Submitting…
                                </>
                            ) : checkingDuplicates ? (
                                <>
                                    <Loader2 className="mr-1.5 size-3.5 animate-spin" /> Checking…
                                </>
                            ) : (
                                `Import ${rows.length} Student${rows.length !== 1 ? "s" : ""}`
                            )}
                        </Button>
                    </div>
                </div>
            )}
        </div>
    );
}
