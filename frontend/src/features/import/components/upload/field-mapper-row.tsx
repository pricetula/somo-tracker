"use client";

import React, { useMemo } from "react";
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Plus, X } from "lucide-react";
import { FieldDef, ExtractedRow } from "./field-mapper";

export interface FieldMapperRowProps {
    field: FieldDef;
    mapping: Record<string, string[] | null>;
    sourceCols: string[];
    extractedRows: ExtractedRow[];
    onChange: (fieldKey: string, value: string | null) => void;
    onRemove: (fieldKey: string, col: string) => void;
}

function getPreview(rows: ExtractedRow[], cols: string[] | null): string[] {
    if (!cols || cols.length === 0) return [];
    const samples: string[] = [];
    for (let i = 0; i < rows.length && samples.length < 3; i++) {
        const values = cols
            .map((c) => {
                const raw = rows[i][c];
                return raw !== undefined && raw !== null && String(raw).trim() !== ""
                    ? String(raw)
                    : "";
            })
            .filter(Boolean);
        const joined = values.join(" ");
        if (joined.trim() !== "") {
            samples.push(joined);
        }
    }
    return samples;
}

export const FieldMapperRow = React.memo(function FieldMapperRow({
    field,
    mapping,
    sourceCols,
    extractedRows,
    onChange,
    onRemove,
}: FieldMapperRowProps) {
    const arr = mapping[field.key] ?? null;
    const isMapped = arr != null && arr.length > 0;
    const preview = useMemo(() => getPreview(extractedRows, arr), [extractedRows, arr]);
    const missingRequired = field.required && !isMapped;
    const available = sourceCols.filter((c) => !arr?.includes(c));

    const handleSelect = (v: string | null) => {
        if (v === "__unmapped__") {
            onChange(field.key, null);
        } else if (v != null) {
            onChange(field.key, v);
        }
    };

    return (
        <div className="mb-2 border-b border-dashed p-4" data-field-key={field.key}>
            <div className="flex items-start justify-between gap-4">
                <div className="min-w-0">
                    <div className="flex items-center gap-2">
                        <span className="text-foreground font-medium tracking-tight">
                            {field.label}
                        </span>
                        {missingRequired && (
                            <span className="text-destructive text-[10px] font-bold tracking-wide uppercase">
                                Required
                            </span>
                        )}
                    </div>
                    {missingRequired && (
                        <p className="text-destructive mt-1">Missing required mapping</p>
                    )}
                </div>
            </div>

            {isMapped ? (
                <div className="mt-3 flex flex-wrap items-center gap-2">
                    {arr!.map((col) => (
                        <Badge
                            key={col}
                            variant="secondary"
                            className="flex max-w-48 items-center gap-1 truncate font-normal"
                        >
                            <span className="truncate">{col}</span>
                            <button
                                type="button"
                                onClick={() => onRemove(field.key, col)}
                                aria-label={`Remove ${col}`}
                                className="hover:bg-muted-foreground/20 ml-0.5 rounded-full p-0.5"
                            >
                                <X className="h-3 w-3" />
                            </button>
                        </Badge>
                    ))}
                    <DropdownMenu>
                        <DropdownMenuTrigger
                            render={
                                <Button
                                    type="button"
                                    variant="ghost"
                                    size="icon"
                                    className="h-6 w-6"
                                    aria-label="Add column"
                                >
                                    <Plus className="h-3 w-3" />
                                </Button>
                            }
                        />
                        <DropdownMenuContent align="start">
                            {available.map((col) => (
                                <DropdownMenuItem key={col} onClick={() => handleSelect(col)}>
                                    {col}
                                </DropdownMenuItem>
                            ))}
                        </DropdownMenuContent>
                    </DropdownMenu>
                </div>
            ) : (
                <div className="mt-3">
                    <DropdownMenu>
                        <DropdownMenuTrigger
                            render={
                                <Button
                                    type="button"
                                    variant="outline"
                                    size="sm"
                                    aria-label="Choose column"
                                >
                                    <Plus className="h-3 w-3" /> <span>Choose column</span>
                                </Button>
                            }
                        />
                        <DropdownMenuContent>
                            <DropdownMenuItem onClick={() => handleSelect("__unmapped__")}>
                                Unmapped
                            </DropdownMenuItem>
                            {sourceCols.map((col) => (
                                <DropdownMenuItem key={col} onClick={() => handleSelect(col)}>
                                    {col}
                                </DropdownMenuItem>
                            ))}
                        </DropdownMenuContent>
                    </DropdownMenu>
                </div>
            )}

            {isMapped && preview.length > 0 && (
                <div className="mt-3 flex flex-wrap items-center gap-2">
                    {preview.map((s, i) => (
                        <Badge
                            key={`${field.key}-${i}-${s}`}
                            variant="outline"
                            className="max-w-48 truncate font-normal"
                        >
                            {String(s).slice(0, 28)}
                            {String(s).length > 28 ? "…" : ""}
                        </Badge>
                    ))}
                </div>
            )}

            {!isMapped && (
                <div className="mt-3 flex flex-wrap items-center gap-2">
                    <span className="text-muted-foreground">No sample values</span>
                </div>
            )}
        </div>
    );
});
