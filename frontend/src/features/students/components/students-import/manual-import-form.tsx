/**
 * StudentManualImportForm — bulk-add students via manual entry.
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

import { submitStudentImport } from "@/lib/api/imports";
import type { ImportRow } from "@/lib/api/imports";
import { getErrorMessage } from "@/lib/errors";

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

// ─── Props ────────────────────────────────────────────────────────────────

interface StudentManualImportFormProps {
    onReset: () => void;
    onJobCreated: (jobId: string, totalRecords: number) => void;
}

// ─── Component ────────────────────────────────────────────────────────────

export function StudentManualImportForm({ onReset, onJobCreated }: StudentManualImportFormProps) {
    const [rows, setRows] = React.useState<StudentRow[]>([freshRow()]);
    const [submitting, setSubmitting] = React.useState(false);
    const submittingRef = React.useRef(false);
    const [fieldErrors, setFieldErrors] = React.useState<Record<string, Record<string, string[]>>>(
        {}
    );

    // ── Row mutation helpers ───────────────────────────────────────────
    const updateRow = React.useCallback((rowId: string, patch: Partial<StudentRow>) => {
        setRows((prev) => prev.map((r) => (r.id === rowId ? { ...r, ...patch } : r)));
    }, []);

    const removeRow = React.useCallback((rowId: string) => {
        setRows((prev) => prev.filter((r) => r.id !== rowId));
    }, []);

    const addRow = React.useCallback(() => {
        setRows((prev) => [...prev, freshRow()]);
    }, []);

    // ── Validation + Submit ────────────────────────────────────────────
    const handleImport = React.useCallback(async () => {
        if (submittingRef.current) return;

        // Validate
        const errors: Record<string, Record<string, string[]>> = {};
        let valid = true;
        rows.forEach((row) => {
            if (!row.fullName.trim()) {
                errors[row.id] = { full_name: ["Full name is required"] };
                valid = false;
            }
        });
        setFieldErrors(errors);
        if (!valid) return;

        submittingRef.current = true;
        setSubmitting(true);

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
            const result = await submitStudentImport({ rows: importRows });
            onJobCreated(result.job_id, importRows.length);
        } catch (err) {
            submittingRef.current = false;
            setSubmitting(false);
            toast.error(getErrorMessage(err));
        }
    }, [rows, onJobCreated]);

    const isSubmitting = submitting;

    return (
        <div className="flex flex-col gap-4">
            {/* Header */}
            <div className="flex items-center justify-between">
                <h3 className="text-sm font-medium">Manual Student Import</h3>
                <div className="flex items-center gap-2">
                    <Button variant="ghost" size="sm" onClick={addRow} disabled={isSubmitting}>
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
                    {rows.map((row, index) => (
                        <div key={row.id} className="bg-muted/30 rounded-md p-3">
                            <div className="mb-2 flex items-center justify-between">
                                <span className="text-muted-foreground text-xs font-medium">
                                    Student {index + 1}
                                </span>
                                <Button
                                    variant="ghost"
                                    size="icon-xs"
                                    onClick={() => removeRow(row.id)}
                                    disabled={isSubmitting || rows.length === 1}
                                    className="text-muted-foreground hover:text-destructive size-6"
                                    aria-label={`Remove row ${index + 1}`}
                                >
                                    <Trash2 className="size-3.5" />
                                </Button>
                            </div>

                            {/* Full Name + Gender */}
                            <div className="mb-2 grid grid-cols-1 gap-2 sm:grid-cols-2">
                                <div className="space-y-1">
                                    <Label className="text-[0.625rem]" htmlFor={`name-${row.id}`}>
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
                                    {fieldErrors[row.id]?.full_name && (
                                        <p className="text-destructive text-[0.625rem]">
                                            {fieldErrors[row.id].full_name[0]}
                                        </p>
                                    )}
                                </div>
                                <div className="space-y-1">
                                    <Label className="text-[0.625rem]" htmlFor={`gender-${row.id}`}>
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
                                    <Label className="text-[0.625rem]" htmlFor={`dob-${row.id}`}>
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
                                    <Label className="text-[0.625rem]" htmlFor={`upi-${row.id}`}>
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
                                        className="h-8 text-xs"
                                    />
                                </div>
                            </div>

                            {/* KNEC + Admission */}
                            <div className="mb-2 grid grid-cols-1 gap-2 sm:grid-cols-2">
                                <div className="space-y-1">
                                    <Label className="text-[0.625rem]" htmlFor={`knec-${row.id}`}>
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
                                        className="h-8 text-xs"
                                    />
                                </div>
                                <div className="space-y-1">
                                    <Label className="text-[0.625rem]" htmlFor={`adm-${row.id}`}>
                                        Admission Number
                                    </Label>
                                    <Input
                                        id={`adm-${row.id}`}
                                        value={row.admissionNumber}
                                        onChange={(e) =>
                                            updateRow(row.id, { admissionNumber: e.target.value })
                                        }
                                        placeholder="e.g. ADM001"
                                        disabled={isSubmitting}
                                        className="h-8 text-xs"
                                    />
                                </div>
                            </div>

                            {/* Class selector */}
                            <div className="space-y-1">
                                <Label className="text-[0.625rem]" htmlFor={`class-${row.id}`}>
                                    Class
                                </Label>
                                <ClassCombobox
                                    value={row.classId}
                                    onChange={(v) => updateRow(row.id, { classId: v as string })}
                                    placeholder="None (no enrollment)"
                                    className="h-8"
                                />
                                <p className="text-muted-foreground text-[0.625rem]">
                                    Leave blank to create without enrollment
                                </p>
                            </div>
                        </div>
                    ))}
                </div>
            )}

            {/* Footer */}
            {rows.length > 0 && (
                <div className="flex items-center justify-between pt-1">
                    <div className="text-muted-foreground text-[0.625rem]">
                        {rows.length} student{rows.length !== 1 ? "s" : ""} ready
                    </div>
                    <div className="flex items-center gap-2">
                        <Button
                            variant="outline"
                            size="sm"
                            onClick={addRow}
                            disabled={isSubmitting}
                        >
                            <Plus className="mr-1 size-3.5" /> Add Row
                        </Button>
                        <Button
                            size="sm"
                            onClick={handleImport}
                            disabled={isSubmitting || rows.length === 0}
                        >
                            {submitting ? (
                                <>
                                    <Loader2 className="mr-1.5 size-3.5 animate-spin" /> Submitting…
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
