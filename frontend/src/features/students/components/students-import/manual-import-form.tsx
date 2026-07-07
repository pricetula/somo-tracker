/**
 * ManualImportForm — bulk-add students via stacked form cards.
 *
 * Each card represents a CreateStudentPayload (with class_id selected via
 * ClassCombobox). Users can add/remove students, then submit all at once.
 */

"use client";

import * as React from "react";
import { Plus, Trash2, Loader2 } from "lucide-react";
import { toast } from "sonner";
import { useMutation } from "@tanstack/react-query";

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
import { createStudents } from "@/lib/api/students";
import type { CreateStudentPayload, CreateStudentsResponse } from "@/lib/api/students";
import { getErrorMessage } from "@/lib/errors";

// ─── Types ────────────────────────────────────────────────────────────────

interface StudentRow {
    /** Stable local id for React keys and deletion tracking. */
    id: string;
    data: CreateStudentPayload;
}

let rowCounter = 0;
function nextRowId() {
    rowCounter += 1;
    return `row-${rowCounter}-${Date.now()}`;
}

function emptyRow(): StudentRow {
    return {
        id: nextRowId(),
        data: {
            full_name: "",
            gender: undefined,
            date_of_birth: null,
            upi_number: null,
            knec_assessment_number: null,
            class_id: null,
        },
    };
}

// ─── Props ────────────────────────────────────────────────────────────────

interface ManualImportFormProps {
    onReset: () => void;
}

// ─── Component ────────────────────────────────────────────────────────────

export function ManualImportForm({ onReset }: ManualImportFormProps) {
    const [rows, setRows] = React.useState<StudentRow[]>([emptyRow()]);
    const [fieldErrors, setFieldErrors] = React.useState<Record<string, Record<string, string[]>>>(
        {}
    );

    const [firstErrorId, setFirstErrorId] = React.useState<string | null>(null);

    // ── Batch submit mutation ──────────────────────────────────────────
    const batchMutation = useMutation({
        mutationFn: async (payloads: CreateStudentPayload[]): Promise<CreateStudentsResponse> => {
            return createStudents({ students: payloads });
        },
        onSuccess: (result, payloads) => {
            const successCount = result.ids.length;
            const failCount = payloads.length - successCount;

            if (successCount > 0) {
                toast.success(`${successCount} student${successCount !== 1 ? "s" : ""} created`);
            }
            if (failCount > 0) {
                toast.error(`${failCount} student${failCount !== 1 ? "s" : ""} failed`);
            }

            // All succeeded — reset form
            setRows([emptyRow()]);
            rowCounter = 0;
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });

    // ── Row mutation helpers ───────────────────────────────────────────
    const updateRow = React.useCallback((rowId: string, patch: Partial<CreateStudentPayload>) => {
        setFirstErrorId(null);
        setRows((prev) =>
            prev.map((row) =>
                row.id === rowId ? { ...row, data: { ...row.data, ...patch } } : row
            )
        );
    }, []);

    const removeRow = React.useCallback((rowId: string) => {
        setRows((prev) => prev.filter((r) => r.id !== rowId));
    }, []);

    const addRow = React.useCallback(() => {
        setRows((prev) => [emptyRow(), ...prev]);
    }, []);

    const validate = React.useCallback((): boolean => {
        const errors: Record<string, Record<string, string[]>> = {};
        let valid = true;
        let firstId: string | null = null;

        rows.forEach((row) => {
            const rowErrors: Record<string, string[]> = {};
            if (!row.data.full_name.trim()) {
                rowErrors.full_name = ["Full name is required"];
                valid = false;
            }
            if (Object.keys(rowErrors).length > 0) {
                errors[row.id] = rowErrors;
                if (firstId === null) firstId = row.id;
            }
        });

        setFieldErrors(errors);
        setFirstErrorId(firstId);
        return valid;
    }, [rows]);

    // Scroll to the first errored card whenever it changes
    React.useEffect(() => {
        if (firstErrorId) {
            const el = document.getElementById(`student-card-${firstErrorId}`);
            el?.scrollIntoView({ behavior: "smooth", block: "center" });
        }
    }, [firstErrorId]);

    // ── Submit ─────────────────────────────────────────────────────────
    const handleImport = React.useCallback(() => {
        if (!validate()) return;
        const payloads = rows.map((r) => ({
            ...r.data,
            full_name: r.data.full_name.trim(),
            gender: r.data.gender || undefined,
            date_of_birth: r.data.date_of_birth || null,
            upi_number: r.data.upi_number || null,
            knec_assessment_number: r.data.knec_assessment_number || null,
            class_id: r.data.class_id || null,
        }));
        batchMutation.mutate(payloads);
    }, [rows, validate, batchMutation]);

    const isSubmitting = batchMutation.isPending;

    // ── Render ─────────────────────────────────────────────────────────
    return (
        <div className="gap-4">
            <div className="flex items-center justify-between pb-4">
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

            {rows.length === 0 ? (
                <div className="text-muted-foreground flex flex-col items-center gap-3 py-12 text-xs">
                    <p>No students added yet.</p>
                    <Button variant="outline" size="sm" onClick={addRow}>
                        <Plus className="mr-1 size-3.5" />
                        Add a student
                    </Button>
                </div>
            ) : (
                <>
                    {/* ── Rows ── stacked guided layout ──────────── */}
                    <div className="flex max-h-100 max-w-full flex-col gap-4 overflow-auto pt-4 pb-4">
                        {rows.map((row, index) => (
                            <div
                                key={row.id}
                                id={`student-card-${row.id}`}
                                className="bg-card space-y-3 rounded-lg border p-4 text-sm"
                            >
                                {/* Full Name — full width */}
                                <div>
                                    <Label
                                        htmlFor={`name-${row.id}`}
                                        className="mb-1.5 block text-xs font-medium"
                                    >
                                        Full Name
                                    </Label>
                                    <Input
                                        id={`name-${row.id}`}
                                        value={row.data.full_name}
                                        onChange={(e) =>
                                            updateRow(row.id, { full_name: e.target.value })
                                        }
                                        placeholder="e.g. John Kiprop"
                                        disabled={isSubmitting}
                                        className="h-9 text-sm"
                                    />
                                    {fieldErrors[row.id]?.full_name && (
                                        <p className="text-destructive mt-1 text-xs">
                                            {fieldErrors[row.id].full_name[0]}
                                        </p>
                                    )}
                                </div>

                                {/* Gender | Date of Birth */}
                                <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                                    <div>
                                        <Label className="mb-1.5 block text-xs font-medium">
                                            Gender
                                        </Label>
                                        <Select
                                            value={row.data.gender ?? ""}
                                            onValueChange={(v) => updateRow(row.id, { gender: v })}
                                            disabled={isSubmitting}
                                        >
                                            <SelectTrigger className="h-9 text-sm">
                                                <SelectValue placeholder="-" />
                                            </SelectTrigger>
                                            <SelectContent>
                                                <SelectItem value="M">Male</SelectItem>
                                                <SelectItem value="F">Female</SelectItem>
                                            </SelectContent>
                                        </Select>
                                    </div>
                                    <div>
                                        <Label className="mb-1.5 block text-xs font-medium">
                                            Date of Birth
                                        </Label>
                                        <DatePicker
                                            value={row.data.date_of_birth ?? ""}
                                            onChange={(v) =>
                                                updateRow(row.id, {
                                                    date_of_birth: v || null,
                                                })
                                            }
                                            placeholder="-"
                                            disabled={isSubmitting}
                                        />
                                    </div>
                                </div>

                                {/* UPI Number | KNEC Number */}
                                <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                                    <div>
                                        <Label className="mb-1.5 block text-xs font-medium">
                                            UPI Number
                                        </Label>
                                        <Input
                                            value={row.data.upi_number ?? ""}
                                            onChange={(e) =>
                                                updateRow(row.id, {
                                                    upi_number: e.target.value || null,
                                                })
                                            }
                                            placeholder="e.g. UP123456789"
                                            disabled={isSubmitting}
                                            className="h-9 text-sm"
                                        />
                                    </div>
                                    <div>
                                        <Label className="mb-1.5 block text-xs font-medium">
                                            KNEC Number
                                        </Label>
                                        <Input
                                            value={row.data.knec_assessment_number ?? ""}
                                            onChange={(e) =>
                                                updateRow(row.id, {
                                                    knec_assessment_number: e.target.value || null,
                                                })
                                            }
                                            placeholder="e.g. KNEC123456"
                                            disabled={isSubmitting}
                                            className="h-9 text-sm"
                                        />
                                    </div>
                                </div>

                                {/* Class | [Delete] */}
                                <div className="flex items-end gap-3">
                                    <div className="flex-1">
                                        <Label className="mb-1.5 block text-xs font-medium">
                                            Class
                                        </Label>
                                        <ClassCombobox
                                            value={row.data.class_id ?? ""}
                                            onChange={(v) =>
                                                updateRow(row.id, {
                                                    class_id: (v as string) || null,
                                                })
                                            }
                                            placeholder="-"
                                            className="h-9"
                                        />
                                    </div>
                                    <Button
                                        variant="outline"
                                        size="icon"
                                        onClick={() => removeRow(row.id)}
                                        disabled={isSubmitting || rows.length === 1}
                                        className="text-muted-foreground hover:text-destructive hover:border-destructive size-9 shrink-0"
                                        aria-label={`Remove student ${index + 1}`}
                                    >
                                        <Trash2 className="size-4" />
                                    </Button>
                                </div>
                            </div>
                        ))}
                    </div>

                    {/* ── Footer ───────────────────────────────── */}
                    <div className="flex items-center justify-between pt-1">
                        <div className="text-muted-foreground text-[0.625rem]">
                            {rows.length} student{rows.length !== 1 ? "s" : ""} ready to import
                        </div>
                        <div className="flex items-center gap-2">
                            <Button
                                size="sm"
                                onClick={handleImport}
                                disabled={isSubmitting || rows.length === 0}
                            >
                                {isSubmitting ? (
                                    <>
                                        <Loader2 className="mr-1.5 size-3.5 animate-spin" />
                                        Importing…
                                    </>
                                ) : (
                                    `Import ${rows.length} Student${rows.length !== 1 ? "s" : ""}`
                                )}
                            </Button>
                        </div>
                    </div>
                </>
            )}
        </div>
    );
}
