/**
 * Phase 4: High-performance virtualized validation motor.
 *
 * Uses TanStack Virtual for smooth rendering of large datasets.
 * Supports dual view toggle (Errors & Warnings / All Records).
 * Editable cells with per-row validation and duplicate handling.
 */

"use client";

import * as React from "react";
import { AlertCircle, AlertTriangle, CheckCircle } from "lucide-react";
import { useVirtualizer } from "@tanstack/react-virtual";
import { Input } from "@/components/ui/input";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { Checkbox } from "@/components/ui/checkbox";
import type { StagedStudentRecord } from "../types";

// ─── Types ─────────────────────────────────────────────────────────────────

interface ValidationMotorProps {
    records: StagedStudentRecord[];
    viewFilter: "errors" | "all";
    onViewFilterChange: (filter: "errors" | "all") => void;
    onUpdateRecord: (rowIndex: number, update: Partial<StagedStudentRecord>) => void;
    onToggleImportAnyway: (rowIndex: number) => void;
    errorCount: number;
    duplicateWarningCount: number;
    totalErrorCount: number;
    submitting: boolean;
    onSubmit: () => void;
    onBack: () => void;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function ValidationMotor({
    records,
    viewFilter,
    onViewFilterChange,
    onUpdateRecord,
    onToggleImportAnyway,
    errorCount,
    duplicateWarningCount,
    submitting,
    onSubmit,
    onBack,
}: ValidationMotorProps) {
    const parentRef = React.useRef<HTMLDivElement>(null);

    // eslint-disable-next-line react-hooks/incompatible-library
    const virtualizer = useVirtualizer({
        count: records.length,
        getScrollElement: () => parentRef.current,
        estimateSize: () => 48,
        overscan: 10,
    });

    const canSubmit = errorCount === 0;

    return (
        <div className="space-y-3">
            {/* Top control bar */}
            <div className="flex items-center justify-between">
                {/* View toggle */}
                <div className="bg-muted flex items-center gap-1 rounded-md p-0.5">
                    <button
                        onClick={() => onViewFilterChange("errors")}
                        className={`rounded px-2.5 py-1 text-xs font-medium transition-colors ${
                            viewFilter === "errors"
                                ? "bg-background text-foreground"
                                : "text-muted-foreground hover:text-foreground"
                        }`}
                    >
                        Errors &amp; Warnings
                    </button>
                    <button
                        onClick={() => onViewFilterChange("all")}
                        className={`rounded px-2.5 py-1 text-xs font-medium transition-colors ${
                            viewFilter === "all"
                                ? "bg-background text-foreground"
                                : "text-muted-foreground hover:text-foreground"
                        }`}
                    >
                        All Records
                    </button>
                </div>

                {/* Stats */}
                <div className="flex items-center gap-3 text-xs">
                    {errorCount > 0 && (
                        <span className="text-destructive flex items-center gap-1">
                            <AlertCircle className="size-3.5" />
                            {errorCount} error{errorCount !== 1 ? "s" : ""}
                        </span>
                    )}
                    {duplicateWarningCount > 0 && (
                        <span className="flex items-center gap-1 text-emerald-600">
                            <AlertTriangle className="size-3.5" />
                            {duplicateWarningCount} duplicate
                            {duplicateWarningCount !== 1 ? "s" : ""}
                        </span>
                    )}
                    <span className="text-muted-foreground">
                        {records.length} record{records.length !== 1 ? "s" : ""}
                    </span>
                </div>
            </div>

            {/* Column headers */}
            <div
                className="text-muted-foreground grid gap-2 px-1 text-xs font-medium"
                style={{
                    gridTemplateColumns: "2fr 70px 110px 1fr 1fr 1fr 1fr 50px",
                }}
            >
                <span>Name</span>
                <span>Gender</span>
                <span>DOB</span>
                <span>UPI</span>
                <span>KNEC</span>
                <span>Parent</span>
                <span>Class</span>
                <span />
            </div>

            {/* Virtualized rows */}
            <div ref={parentRef} className="max-h-160 overflow-auto">
                <div
                    style={{
                        height: `${virtualizer.getTotalSize()}px`,
                        width: "100%",
                        position: "relative",
                    }}
                >
                    {virtualizer.getVirtualItems().map((virtualItem) => {
                        const record = records[virtualItem.index];
                        const hasErrors = !record.isValid;
                        const isDuplicate = record.isDuplicate && !record.importAnyway;
                        const hasAdvisories = Object.keys(record.advisories).length > 0;

                        let rowBorderClass = "";
                        if (isDuplicate) rowBorderClass = "border-l-2 border-l-emerald-600";
                        else if (hasErrors) rowBorderClass = "border-l-2 border-l-destructive";
                        else if (record.isValid) rowBorderClass = "border-l-2 border-l-emerald-500";

                        const rowOpacity = record.isValid && !isDuplicate ? "opacity-70" : "";

                        return (
                            <div
                                key={record._rowIndex}
                                data-index={virtualItem.index}
                                ref={virtualizer.measureElement}
                                className={`absolute top-0 left-0 w-full ${rowBorderClass} ${rowOpacity}`}
                                style={{
                                    transform: `translateY(${virtualItem.start}px)`,
                                }}
                            >
                                <div
                                    className="grid gap-2 px-1 py-1"
                                    style={{
                                        gridTemplateColumns: "2fr 70px 110px 1fr 1fr 1fr 1fr 50px",
                                    }}
                                >
                                    {/* Name */}
                                    <CellWrapper
                                        hasError={!!record.errors.full_name}
                                        advisory={!!record.advisories.full_name}
                                    >
                                        <Input
                                            value={record.full_name}
                                            onChange={(e) =>
                                                onUpdateRecord(record._rowIndex, {
                                                    full_name: e.target.value,
                                                })
                                            }
                                            className={`h-8 text-sm ${record.errors.full_name ? "border-destructive ring-destructive/30 ring-1" : ""} ${record.advisories.full_name ? "border-emerald-500/50" : ""}`}
                                        />
                                    </CellWrapper>

                                    {/* Gender */}
                                    <CellWrapper hasError={!!record.errors.gender}>
                                        <Select
                                            value={record.gender ?? ""}
                                            onValueChange={(v) =>
                                                onUpdateRecord(record._rowIndex, {
                                                    gender: v === "M" || v === "F" ? v : null,
                                                    errors: {
                                                        ...record.errors,
                                                        gender: v ? "" : "Gender is required",
                                                    },
                                                    isValid: !record.errors.full_name && !!v,
                                                } as Partial<StagedStudentRecord>)
                                            }
                                        >
                                            <SelectTrigger
                                                className={`h-8 text-sm ${record.errors.gender ? "border-destructive ring-destructive/30 ring-1" : ""}`}
                                            >
                                                <SelectValue placeholder="-" />
                                            </SelectTrigger>
                                            <SelectContent>
                                                <SelectItem value="M">M</SelectItem>
                                                <SelectItem value="F">F</SelectItem>
                                            </SelectContent>
                                        </Select>
                                    </CellWrapper>

                                    {/* DOB */}
                                    <CellWrapper
                                        hasError={!!record.errors.date_of_birth}
                                        advisory={!!record.advisories.date_of_birth}
                                    >
                                        <Input
                                            value={record.date_of_birth ?? ""}
                                            onChange={(e) =>
                                                onUpdateRecord(record._rowIndex, {
                                                    date_of_birth: e.target.value || null,
                                                })
                                            }
                                            className={`h-8 text-sm ${record.errors.date_of_birth ? "border-destructive ring-destructive/30 ring-1" : ""} ${record.advisories.date_of_birth ? "border-emerald-500/50" : ""}`}
                                            placeholder="YYYY-MM-DD"
                                        />
                                    </CellWrapper>

                                    {/* UPI */}
                                    <CellWrapper hasError={!!record.errors.upi_number}>
                                        <Input
                                            value={record.upi_number ?? ""}
                                            onChange={(e) =>
                                                onUpdateRecord(record._rowIndex, {
                                                    upi_number: e.target.value || null,
                                                })
                                            }
                                            className={`h-8 text-sm ${record.errors.upi_number ? "border-destructive ring-destructive/30 ring-1" : ""}`}
                                        />
                                    </CellWrapper>

                                    {/* KNEC */}
                                    <CellWrapper hasError={!!record.errors.knec_assessment_number}>
                                        <Input
                                            value={record.knec_assessment_number ?? ""}
                                            onChange={(e) =>
                                                onUpdateRecord(record._rowIndex, {
                                                    knec_assessment_number: e.target.value || null,
                                                })
                                            }
                                            className={`h-8 text-sm ${record.errors.knec_assessment_number ? "border-destructive ring-destructive/30 ring-1" : ""}`}
                                        />
                                    </CellWrapper>

                                    {/* Parent */}
                                    <CellWrapper advisory={!!record.advisories.parent}>
                                        <div className="relative">
                                            <Input
                                                value={record.cbc_student_parents_id ?? ""}
                                                readOnly
                                                className="text-muted-foreground h-8 text-sm"
                                                placeholder={
                                                    record.advisories.parent
                                                        ? "Not found"
                                                        : "Auto-matched"
                                                }
                                            />
                                        </div>
                                    </CellWrapper>

                                    {/* Class */}
                                    <CellWrapper advisory={!!record.advisories.class}>
                                        <div className="relative">
                                            <Input
                                                value={record.class_id ?? ""}
                                                readOnly
                                                className="text-muted-foreground h-8 text-sm"
                                                placeholder={
                                                    record.advisories.class
                                                        ? "Not found"
                                                        : "Auto-matched"
                                                }
                                            />
                                        </div>
                                    </CellWrapper>

                                    {/* Actions: duplicate override checkbox */}
                                    <div className="flex items-center justify-center">
                                        {isDuplicate && (
                                            <div className="flex items-center gap-1">
                                                <Checkbox
                                                    id={`import-anyway-${record._rowIndex}`}
                                                    checked={record.importAnyway}
                                                    onCheckedChange={() =>
                                                        onToggleImportAnyway(record._rowIndex)
                                                    }
                                                />
                                                <label
                                                    htmlFor={`import-anyway-${record._rowIndex}`}
                                                    className="text-muted-foreground cursor-pointer text-[10px] leading-tight"
                                                >
                                                    Import
                                                    <br />
                                                    anyway
                                                </label>
                                            </div>
                                        )}
                                        {!isDuplicate && record.isValid && (
                                            <CheckCircle className="size-4 text-emerald-500" />
                                        )}
                                    </div>
                                </div>

                                {/* Inline advisory text */}
                                {(isDuplicate || hasAdvisories) && (
                                    <div className="flex items-center gap-2 px-1 pb-1">
                                        {isDuplicate && (
                                            <p className="text-[10px] text-emerald-600">
                                                Possible duplicate: matches existing student.
                                            </p>
                                        )}
                                        {record.advisories.parent && (
                                            <p className="text-[10px] text-emerald-600">
                                                {record.advisories.parent}
                                            </p>
                                        )}
                                        {record.advisories.class && (
                                            <p className="text-[10px] text-emerald-600">
                                                {record.advisories.class}
                                            </p>
                                        )}
                                        {record.advisories.date_of_birth && (
                                            <p className="text-[10px] text-emerald-600">
                                                {record.advisories.date_of_birth}
                                            </p>
                                        )}
                                    </div>
                                )}
                            </div>
                        );
                    })}
                </div>
            </div>

            {/* Bottom actions */}
            <div className="flex items-center justify-between pt-2">
                <button
                    onClick={onBack}
                    className="text-muted-foreground hover:text-foreground text-sm"
                    disabled={submitting}
                >
                    Back
                </button>

                <div className="flex items-center gap-3">
                    <button
                        onClick={onSubmit}
                        disabled={!canSubmit || submitting}
                        className="bg-primary text-primary-foreground hover:bg-primary/90 rounded-md px-5 py-1.5 text-sm font-medium disabled:opacity-50"
                    >
                        {submitting ? "Submitting…" : `Submit Import (${records.length})`}
                    </button>
                </div>
            </div>
        </div>
    );
}

// ─── Cell Wrapper (adds error/warning indicators) ─────────────────────────

function CellWrapper({
    hasError,
    advisory,
    children,
}: {
    hasError?: boolean;
    advisory?: boolean;
    children: React.ReactNode;
}) {
    return (
        <div className="relative">
            {children}
            {hasError && (
                <AlertCircle className="text-destructive absolute top-1/2 right-2 size-3.5 -translate-y-1/2" />
            )}
            {advisory && !hasError && (
                <AlertTriangle className="absolute top-1/2 right-2 size-3.5 -translate-y-1/2 text-emerald-600" />
            )}
        </div>
    );
}
