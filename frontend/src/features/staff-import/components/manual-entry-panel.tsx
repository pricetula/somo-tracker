/**
 * Manual Entry Panel — dynamic inline repeating form rows for bulk staff entry.
 *
 * Validates rows in real-time using TanStack Virtual for smooth rendering at scale.
 * Critical errors block submission; warnings & auto-corrections are surfaced inline.
 */

"use client";

import * as React from "react";
import { X, Plus, AlertCircle, PhoneOff } from "lucide-react";

import { saveDraft, type ImportDraftRow } from "@/lib/db";
import type { AllowedRole } from "./bulk-staff-import-dialog";
import { ImportInput } from "./import-input";
import { useVirtualizer, hasValidEmailStructure, normalizePhone } from "../lib/validation";

// ─── Types ─────────────────────────────────────────────────────────────────

interface ManualEntryPanelProps {
    onRowsReady: (rows: ImportDraftRow[]) => void;
    role: AllowedRole;
    tenantID: string;
    userID: string;
    context: string;
}

interface RowValidation {
    emailError: boolean;
    nameError: boolean;
    phoneWarning: boolean;
    phoneNormalized: string | null;
    duplicateError: boolean;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function ManualEntryPanel({
    onRowsReady,
    role,
    tenantID,
    userID,
    context,
}: ManualEntryPanelProps) {
    const parentRef = React.useRef<HTMLDivElement>(null);
    const [rows, setRows] = React.useState<ImportDraftRow[]>([
        {
            temp_id: crypto.randomUUID(),
            email: "",
            full_name: "",
            phone: "",
            registration_number: "",
        },
    ]);

    // Track auto-corrected cells for visual highlighting
    const [correctedCells, setCorrectedCells] = React.useState<Set<string>>(new Set());

    const isTeacher = role === "TEACHER";

    const virtualizer = useVirtualizer({
        count: rows.length,
        getScrollElement: () => parentRef.current,
        estimateSize: () => 48,
        overscan: 10,
    });

    // Save draft on change
    React.useEffect(() => {
        if (rows.length > 0) {
            saveDraft(tenantID, userID, context, rows);
        }
    }, [rows, tenantID, userID, context]);

    function addRow() {
        setRows((prev) => [
            ...prev,
            {
                temp_id: crypto.randomUUID(),
                email: "",
                full_name: "",
                phone: "",
                registration_number: "",
            },
        ]);
    }

    function removeRow(tempID: string) {
        setRows((prev) => prev.filter((r) => r.temp_id !== tempID));
    }

    function updateRow(tempID: string, field: keyof ImportDraftRow, value: string) {
        setRows((prev) => prev.map((r) => (r.temp_id !== tempID ? r : { ...r, [field]: value })));
    }

    function handlePhoneBlur(tempID: string, e: React.FocusEvent<HTMLInputElement>) {
        const value = e.target.value.trim();
        if (!value) return;

        const normalized = normalizePhone(value);
        if (normalized && normalized !== value) {
            setRows((prev) =>
                prev.map((r) => (r.temp_id !== tempID ? r : { ...r, phone: normalized }))
            );
            setCorrectedCells((prev) => new Set(prev).add(`${tempID}:phone`));
        }
    }
    // Compute validation
    const emailCounts = React.useMemo(() => {
        const counts = new Map<string, number>();
        for (const row of rows) {
            const lower = row.email.toLowerCase().trim();
            if (lower) {
                counts.set(lower, (counts.get(lower) || 0) + 1);
            }
        }
        return counts;
    }, [rows]);

    const validations = React.useMemo(() => {
        return rows.map((row): RowValidation => {
            const email = row.email.trim();
            const lowerEmail = email.toLowerCase();

            const hasEmail = email !== "";
            return {
                emailError: hasEmail && !hasValidEmailStructure(email),
                nameError: hasEmail && row.full_name === "",
                phoneWarning: row.phone !== "" && normalizePhone(row.phone) === null,
                phoneNormalized: normalizePhone(row.phone),
                duplicateError: hasEmail && (emailCounts.get(lowerEmail) ?? 0) > 1,
            };
        });
    }, [rows, emailCounts]);

    // Compute TSC Number validation separately (only for TEACHER)
    const tscErrors = React.useMemo(() => {
        if (!isTeacher) return new Set<string>();
        const errors = new Set<string>();
        for (const row of rows) {
            const email = row.email.trim();
            if (!email) continue;
            if (!row.registration_number?.trim()) {
                errors.add(row.temp_id);
            }
        }
        return errors;
    }, [rows, isTeacher]);

    // Count critical errors (include TSC errors for TEACHER)
    const criticalErrorCount = React.useMemo(() => {
        let count = validations.filter(
            (v) => v.emailError || v.nameError || v.duplicateError
        ).length;
        if (isTeacher) {
            count += tscErrors.size;
        }
        return count;
    }, [validations, isTeacher, tscErrors]);

    const emptyRows = rows.filter((r) => !r.email.trim()).length;
    const ready = rows.length > 0 && rows.some((r) => r.email.trim()) && criticalErrorCount === 0;

    function handleProceed() {
        if (!ready) return;
        // Normalize all phone numbers before passing up
        const normalized = rows.map((r) => {
            const phone = normalizePhone(r.phone);
            return { ...r, phone: phone ?? "" };
        });
        onRowsReady(normalized);
    }

    return (
        <div className="flex flex-col gap-3">
            {/* Error summary banner */}
            {criticalErrorCount > 0 && (
                <div className="bg-destructive/10 text-destructive flex items-start gap-2 rounded-md px-3 py-2 text-sm">
                    <AlertCircle className="mt-0.5 size-4 shrink-0" />
                    <span>
                        {criticalErrorCount} critical error{criticalErrorCount !== 1 ? "s" : ""} —
                        fix before submitting.
                        {emptyRows > 0 &&
                            ` ${emptyRows} empty row${emptyRows !== 1 ? "s" : ""} will be skipped.`}
                    </span>
                </div>
            )}
            {/* Column headers */}
            <div
                className="text-muted-foreground grid gap-2 px-1 text-xs font-medium"
                style={{
                    gridTemplateColumns: isTeacher
                        ? "1fr 1.5fr 0.8fr 1fr 28px"
                        : "1fr 1.5fr 1fr 28px",
                }}
            >
                <span>Full Name *</span>
                <span>Email *</span>
                {isTeacher && <span>TSC Number *</span>}
                <span>Phone</span>
                <span />
            </div>

            {/* Virtualized rows */}
            <div ref={parentRef} className="max-h-100 overflow-auto">
                <div
                    style={{
                        height: `${virtualizer.getTotalSize()}px`,
                        width: "100%",
                        position: "relative",
                    }}
                >
                    {virtualizer.getVirtualItems().map((virtualItem) => {
                        const row = rows[virtualItem.index];
                        const val = validations[virtualItem.index];
                        const isPhoneCorrected = correctedCells.has(`${row.temp_id}:phone`);

                        return (
                            <div
                                key={row.temp_id}
                                data-index={virtualItem.index}
                                ref={virtualizer.measureElement}
                                className="absolute top-0 left-0 w-full"
                                style={{
                                    transform: `translateY(${virtualItem.start}px)`,
                                }}
                            >
                                <div
                                    className="grid gap-2 px-1 py-1"
                                    style={{
                                        gridTemplateColumns: isTeacher
                                            ? "1fr 1.5fr 0.8fr 1fr 28px"
                                            : "1fr 1.5fr 1fr 28px",
                                    }}
                                >
                                    <ImportInput
                                        placeholder="Jane Doe"
                                        value={row.full_name}
                                        onChange={(e) =>
                                            updateRow(row.temp_id, "full_name", e.target.value)
                                        }
                                        className={`h-9 text-sm ${val.nameError && row.full_name ? "border-destructive" : ""}`}
                                    />
                                    <div className="relative">
                                        <ImportInput
                                            placeholder="jane@school.edu"
                                            value={row.email}
                                            onChange={(e) =>
                                                updateRow(row.temp_id, "email", e.target.value)
                                            }
                                            className={`h-9 text-sm ${val.emailError || val.duplicateError ? "border-destructive pr-7" : ""}`}
                                        />
                                        {(val.emailError || val.duplicateError) && (
                                            <AlertCircle className="text-destructive absolute top-1/2 right-2 size-4 -translate-y-1/2" />
                                        )}
                                    </div>
                                    {isTeacher && (
                                        <div className="relative">
                                            <ImportInput
                                                placeholder="TSC-XXX-XXX"
                                                value={row.registration_number}
                                                onChange={(e) =>
                                                    updateRow(
                                                        row.temp_id,
                                                        "registration_number",
                                                        e.target.value
                                                    )
                                                }
                                                className={`h-9 text-sm ${tscErrors.has(row.temp_id) ? "border-destructive pr-7" : ""}`}
                                            />
                                            {tscErrors.has(row.temp_id) && (
                                                <AlertCircle className="text-destructive absolute top-1/2 right-2 size-4 -translate-y-1/2" />
                                            )}
                                        </div>
                                    )}
                                    <div className="relative">
                                        <ImportInput
                                            placeholder="+254 712 345 678"
                                            value={row.phone}
                                            onChange={(e) =>
                                                updateRow(row.temp_id, "phone", e.target.value)
                                            }
                                            onBlur={(e) => handlePhoneBlur(row.temp_id, e)}
                                            className={`h-9 text-sm ${isPhoneCorrected || val.phoneWarning ? "pr-7" : ""} ${isPhoneCorrected || val.phoneWarning ? "border-destructive/50" : ""}`}
                                        />
                                        {(isPhoneCorrected || val.phoneWarning) && (
                                            <PhoneOff className="text-destructive absolute top-1/2 right-2 size-4 -translate-y-1/2" />
                                        )}
                                    </div>
                                    <button
                                        onClick={() => removeRow(row.temp_id)}
                                        disabled={rows.length <= 1}
                                        className="text-muted-foreground hover:text-foreground mt-1 flex size-7 items-center justify-center rounded-md disabled:opacity-30"
                                    >
                                        <X className="size-4" />
                                    </button>
                                </div>
                            </div>
                        );
                    })}
                </div>
            </div>

            {/* Add row + proceed */}
            <div className="flex items-center justify-between px-1 pt-2">
                <button
                    onClick={addRow}
                    className="text-muted-foreground hover:text-foreground flex items-center gap-1.5 text-xs font-medium"
                >
                    <Plus className="size-3.5" />
                    Add another
                </button>

                <div className="flex items-center gap-3">
                    <span className="text-muted-foreground text-xs">
                        {rows.filter((r) => r.email.trim()).length} filled
                        {criticalErrorCount > 0 && ` · ${criticalErrorCount} errors`}
                    </span>
                    <button
                        onClick={handleProceed}
                        disabled={!ready}
                        className="bg-primary text-primary-foreground hover:bg-primary/90 rounded-md px-4 py-1.5 text-sm font-medium disabled:opacity-50"
                    >
                        Review & Submit
                    </button>
                </div>
            </div>
        </div>
    );
}
