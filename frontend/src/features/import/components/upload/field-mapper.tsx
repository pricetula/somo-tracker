"use client";

import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { FieldMapperRow } from "./field-mapper-row";

export interface FieldDef {
    key: string;
    label: string;
    aliases: string[];
    required: boolean;
    validator?: (value: unknown, row: Record<string, unknown>) => string | null;
}

export interface ExtractedRow {
    __id: string;
    [column: string]: unknown;
}

export interface MappedRow {
    id: string;
    data: Record<string, unknown>;
    errors?: Record<string, string>;
}

export interface MappingResult {
    mapping: Record<string, string[] | null>;
    rows: MappedRow[];
    isValid: boolean;
}

export interface FieldMapperProps {
    extractedRows: ExtractedRow[];
    desiredFields: FieldDef[];
    onMappingChange: (result: MappingResult) => void;
    isSubmitting?: boolean;
}

function normalize(str: string): string {
    return String(str)
        .toLowerCase()
        .replace(/[^a-z0-9]/g, "");
}

function computeAutoMapping(
    rows: ExtractedRow[],
    fields: FieldDef[]
): Record<string, string[] | null> {
    if (!rows || rows.length === 0) {
        return Object.fromEntries(fields.map((f) => [f.key, null]));
    }
    const sourceCols = Object.keys(rows[0]).filter((k) => k !== "__id");
    const result: Record<string, string[] | null> = {};
    for (const f of fields) {
        const aliases = [f.key, f.label, ...f.aliases].map(String);
        let match: string | null = null;
        for (const col of sourceCols) {
            const normCol = normalize(col);
            for (const a of aliases) {
                const normA = normalize(a);
                if (normCol === normA || normCol.includes(normA) || normA.includes(normCol)) {
                    match = col;
                    break;
                }
            }
            if (match) break;
        }
        result[f.key] = match ? [match] : null;
    }
    return result;
}

function safeValidator(
    validator: NonNullable<FieldDef["validator"]>,
    value: unknown,
    row: Record<string, unknown>
): string | null {
    try {
        return validator(value, row) ?? null;
    } catch {
        return "Validation error";
    }
}

function computeResult(
    rows: ExtractedRow[],
    fields: FieldDef[],
    mapping: Record<string, string[] | null>
): MappingResult {
    const mappingOut: Record<string, string[] | null> = {};
    for (const f of fields) {
        mappingOut[f.key] = mapping[f.key] ?? null;
    }

    const processedRows = rows.map((row) => {
        const data: Record<string, unknown> = {};
        const errors: Record<string, string> = {};
        for (const f of fields) {
            const cols = mapping[f.key];
            if (cols != null && cols.length > 0) {
                const values = cols
                    .map((c) => row[c])
                    .filter((v) => v !== undefined && v !== null && String(v).trim() !== "")
                    .map(String);
                const value = values.join(" ");
                data[f.key] = value || null;
                if (f.validator && value !== "") {
                    const msg = safeValidator(f.validator, value, row);
                    if (msg != null && msg !== "") {
                        errors[f.key] = msg;
                    }
                }
            }
        }
        const base: MappedRow = { id: row.__id as string, data };
        if (Object.keys(errors).length > 0) {
            base.errors = errors;
        }
        return base;
    });

    const requiredMapped = fields.every(
        (f) => !f.required || (mapping[f.key] != null && mapping[f.key]!.length > 0)
    );
    const anyErrors = processedRows.some((r) => "errors" in r);
    const isValid = requiredMapped && !anyErrors;

    return {
        mapping: mappingOut,
        rows: processedRows,
        isValid,
    };
}

function hashRows(rows: ExtractedRow[]): string {
    if (!rows || rows.length === 0) return "empty";
    const ids = rows.map((r) => r.__id ?? "").join("|");
    const firstKeys = Object.keys(rows[0])
        .filter((k) => k !== "__id")
        .slice(0, 5)
        .join(",");
    return `${rows.length}:${ids.slice(0, 200)}:${firstKeys}`;
}

export const FieldMapper = React.memo(function FieldMapper({
    extractedRows,
    desiredFields,
    onMappingChange,
    isSubmitting,
}: FieldMapperProps) {
    const [mapping, setMapping] = useState<Record<string, string[] | null>>(() =>
        computeAutoMapping(extractedRows, desiredFields)
    );

    const prevRowsHashRef = useRef<string>(hashRows(extractedRows));
    const prevFieldsRef = useRef<string>(
        desiredFields.map((f) => `${f.key}:${f.required}`).join(",")
    );

    useEffect(() => {
        const rowsHash = hashRows(extractedRows);
        const fieldsHash = desiredFields.map((f) => `${f.key}:${f.required}`).join(",");
        if (rowsHash !== prevRowsHashRef.current || fieldsHash !== prevFieldsRef.current) {
            prevRowsHashRef.current = rowsHash;
            prevFieldsRef.current = fieldsHash;
            setMapping(computeAutoMapping(extractedRows, desiredFields));
        }
    }, [extractedRows, desiredFields]);

    const sourceCols = useMemo(() => {
        if (!extractedRows || extractedRows.length === 0) return [];
        return Object.keys(extractedRows[0]).filter((k) => k !== "__id");
    }, [extractedRows]);

    const result = useMemo(
        () => computeResult(extractedRows, desiredFields, mapping),
        [extractedRows, desiredFields, mapping]
    );

    const mappedCount = useMemo(
        () =>
            desiredFields.filter((f) => mapping[f.key] != null && mapping[f.key]!.length > 0)
                .length,
        [desiredFields, mapping]
    );
    const errorCount = useMemo(
        () => result.rows.reduce((c, r) => c + (r.errors ? Object.keys(r.errors).length : 0), 0),
        [result]
    );

    const handleAutoMatch = useCallback(() => {
        setMapping(computeAutoMapping(extractedRows, desiredFields));
    }, [extractedRows, desiredFields]);

    const handleChange = useCallback((fieldKey: string, value: string | null) => {
        setMapping((prev) => {
            const current = prev[fieldKey];
            if (value == null || value === "__unmapped__") {
                return { ...prev, [fieldKey]: null };
            }
            if (current == null) {
                return { ...prev, [fieldKey]: [value] };
            }
            if (current.includes(value)) return prev;
            return { ...prev, [fieldKey]: [...current, value] };
        });
    }, []);

    const handleRemoveColumn = useCallback((fieldKey: string, col: string) => {
        setMapping((prev) => {
            const current = prev[fieldKey];
            if (!current) return prev;
            const next = current.filter((c) => c !== col);
            return { ...prev, [fieldKey]: next.length > 0 ? next : null };
        });
    }, []);

    if (!extractedRows || extractedRows.length === 0) {
        return (
            <div className="rounded-xl border border-dashed p-10 text-center">
                <p className="text-foreground font-medium tracking-tight">No rows to map</p>
                <p className="text-muted-foreground mt-1">
                    Upload a file with data to start mapping fields.
                </p>
            </div>
        );
    }

    return (
        <div>
            <div className="bg-muted/40 mb-4 flex items-center justify-between rounded-lg border px-4 py-3">
                <div className="flex items-center gap-4 text-sm">
                    <span className="font-medium">
                        {mappedCount} / {desiredFields.length} mapped
                    </span>
                    <span className="text-muted-foreground">
                        {errorCount} validation errors in preview
                    </span>
                </div>
                <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={handleAutoMatch}
                    aria-label="Reset to auto-match"
                >
                    Auto-match
                </Button>
            </div>

            {desiredFields.map((field) => (
                <FieldMapperRow
                    key={field.key}
                    field={field}
                    mapping={mapping}
                    sourceCols={sourceCols}
                    extractedRows={extractedRows}
                    onChange={handleChange}
                    onRemove={handleRemoveColumn}
                />
            ))}
            <div className="mt-6 flex items-center justify-between gap-4 border-t pt-6">
                <div className="flex items-center gap-3">
                    <span
                        className={`inline-flex h-2 w-2 rounded-full ${
                            result.isValid ? "bg-emerald-500" : "bg-amber-400"
                        }`}
                        aria-hidden="true"
                    />
                    <span className="text-sm font-medium tracking-tight">
                        {result.isValid ? "Ready to save" : "Incomplete mapping"}
                    </span>
                </div>
                <Button
                    type="button"
                    disabled={!result.isValid || isSubmitting}
                    onClick={() => onMappingChange(result)}
                    aria-label="Save mapping"
                >
                    Save mapping
                </Button>
            </div>
        </div>
    );
});
