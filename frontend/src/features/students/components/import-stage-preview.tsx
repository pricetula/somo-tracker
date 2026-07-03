/**
 * Stage 2: Preview & Validation.
 *
 * - Paginated grid (50 rows/page)
 * - Filter toggle: All rows / Rows with errors
 * - Inline class re-selection for rows with invalid_class
 * - Skip mechanism for errored rows
 * - Server-rejected rows (from Stage 4 reconciliation) shown with badges
 * - Progress bar during initial worker processing
 */

"use client";

import * as React from "react";

import { Button } from "@/components/ui/button";
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table";
import { ClassSelector } from "./class-selector";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { Badge } from "@/components/ui/badge";
import { Loader2, SkipForward, Undo2 } from "lucide-react";

import { useImportStore } from "../hooks/use-import-store";
import type { StagedRow, ImportStage } from "@/lib/import-data/types";
import type { ClassMatchRecord } from "@/lib/import-data/matching";
import { listClasses } from "@/lib/api/classes";
import { cn } from "@/lib/utils";

// ─── Props ─────────────────────────────────────────────────────────────────

interface ImportStagePreviewProps {
    onStageChange: (stage: ImportStage) => void;
    onClose: () => void;
}

const PAGE_SIZE = 50;

// ─── Component ─────────────────────────────────────────────────────────────

export function ImportStagePreview({ onStageChange, onClose }: ImportStagePreviewProps) {
    const store = useImportStore();

    const [pageRows, setPageRows] = React.useState<StagedRow[]>([]);
    const [totalFiltered, setTotalFiltered] = React.useState(0);
    const [currentPage, setCurrentPage] = React.useState(1);
    const [filter, setFilter] = React.useState<"all" | "errors">("all");
    const [availableClasses, setAvailableClasses] = React.useState<ClassMatchRecord[]>([]);
    const [loading, setLoading] = React.useState(true);
    const totalPages = Math.max(1, Math.ceil(totalFiltered / PAGE_SIZE));

    // ─── Load classes for inline selection ─────────────────────────────────
    React.useEffect(() => {
        const loadClasses = async () => {
            if (!store.meta) return;
            try {
                const res = await listClasses({
                    academic_year_id: store.meta.academic_year_id,
                    academic_term_id: store.meta.academic_term_id,
                    limit: 500,
                });
                setAvailableClasses(
                    (res.data ?? []).map((c) => ({
                        id: c.id,
                        grade_level: c.grade_level,
                        stream_name: c.stream_name,
                        display_label: c.display_label,
                    }))
                );
            } catch {
                setAvailableClasses([]);
            }
        };
        loadClasses();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    // ─── Load page data ───────────────────────────────────────────────────
    React.useEffect(() => {
        const load = async () => {
            setLoading(true);
            try {
                const result = await store.loadPage(
                    currentPage,
                    PAGE_SIZE,
                    filter === "errors" ? { hasError: true } : undefined
                );
                setPageRows(result.rows);
                setTotalFiltered(result.total);
            } finally {
                setLoading(false);
            }
        };
        load();
    }, [currentPage, filter, store]);

    // ─── Refresh when store updates ───────────────────────────────────────
    React.useEffect(() => {
        const refresh = async () => {
            const result = await store.loadPage(
                currentPage,
                PAGE_SIZE,
                filter === "errors" ? { hasError: true } : undefined
            );
            setPageRows(result.rows);
            setTotalFiltered(result.total);
        };
        refresh();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [store.stagedRows]);

    // ─── Inline class update ──────────────────────────────────────────────
    const handleClassChange = React.useCallback(
        async (rowNumber: number, classId: string) => {
            const cls = availableClasses.find((c) => c.id === classId);
            if (!cls) return;

            await store.updateRow(rowNumber, {
                processed_data: {
                    ...pageRows.find((r) => r.row_number === rowNumber)!.processed_data,
                    class_id: classId,
                    grade_level: cls.grade_level,
                    stream_name: cls.stream_name,
                },
                ui_meta: {
                    ...pageRows.find((r) => r.row_number === rowNumber)!.ui_meta,
                    has_error: false,
                    errors: {
                        ...pageRows.find((r) => r.row_number === rowNumber)!.ui_meta.errors,
                        invalid_class: null,
                    },
                },
            });
        },
        [availableClasses, pageRows, store]
    );

    // ─── Skip / Unskip row ────────────────────────────────────────────────
    const toggleSkip = React.useCallback(
        async (rowNumber: number, currentSkipped: boolean) => {
            await store.updateRow(rowNumber, {
                ui_meta: {
                    ...pageRows.find((r) => r.row_number === rowNumber)!.ui_meta,
                    skipped: !currentSkipped,
                },
            });
        },
        [pageRows, store]
    );

    // ─── Filter change ────────────────────────────────────────────────────
    const handleFilterChange = React.useCallback((value: string) => {
        setFilter(value as "all" | "errors");
        setCurrentPage(1);
    }, []);

    // ─── Continue to Stage 3 ──────────────────────────────────────────────
    const hasUnresolvedErrors =
        store.errorCount > 0 || pageRows.some((r) => r.ui_meta.has_error && !r.ui_meta.skipped);

    const handleContinue = () => {
        if (hasUnresolvedErrors) return;
        store.setStage("READY");
        onStageChange("READY");
    };

    // ─── Render error badge ───────────────────────────────────────────────
    const renderErrorBadge = (row: StagedRow) => {
        const err = row.ui_meta.errors;
        if (err.server_rejected && err.server_error_type) {
            const label = getErrorTypeLabel(err.server_error_type);
            return (
                <Badge variant="destructive" className="text-xs">
                    {label}
                </Badge>
            );
        }
        return null;
    };

    return (
        <div className="flex flex-1 flex-col gap-4 overflow-hidden">
            {/* Summary bar */}
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-4">
                    <p className="text-muted-foreground text-sm">
                        {store.meta?.total_rows ?? 0} total rows
                    </p>
                    {store.errorCount > 0 && (
                        <p className="text-destructive text-sm">
                            {store.errorCount} need attention
                        </p>
                    )}
                    {store.skippedCount > 0 && (
                        <p className="text-muted-foreground text-sm">
                            {store.skippedCount} skipped
                        </p>
                    )}
                </div>

                <ToggleGroup
                    type="single"
                    value={filter}
                    onValueChange={handleFilterChange}
                    size="sm"
                >
                    <ToggleGroupItem value="all">All rows</ToggleGroupItem>
                    <ToggleGroupItem value="errors">
                        Rows with errors
                        {store.errorCount > 0 && (
                            <span className="bg-destructive/10 text-destructive ml-1.5 rounded-full px-1.5 py-0.5 text-xs">
                                {store.errorCount}
                            </span>
                        )}
                    </ToggleGroupItem>
                </ToggleGroup>
            </div>

            {/* Table */}
            <div className="flex-1 overflow-auto rounded-md border">
                <Table>
                    <TableHeader className="bg-muted/30 sticky top-0">
                        <TableRow>
                            <TableHead className="w-12">#</TableHead>
                            <TableHead>Full Name</TableHead>
                            <TableHead className="w-16">Gender</TableHead>
                            <TableHead className="w-28">Date of Birth</TableHead>
                            <TableHead className="w-40">Class</TableHead>
                            <TableHead className="w-28">NEMIS</TableHead>
                            <TableHead className="w-28">Assessment</TableHead>
                            <TableHead className="w-20">Errors</TableHead>
                            <TableHead className="w-16">Action</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {loading ? (
                            <TableRow>
                                <TableCell colSpan={9} className="py-8 text-center">
                                    <Loader2 className="text-muted-foreground mx-auto size-5 animate-spin" />
                                </TableCell>
                            </TableRow>
                        ) : pageRows.length === 0 ? (
                            <TableRow>
                                <TableCell
                                    colSpan={9}
                                    className="text-muted-foreground py-8 text-center text-sm"
                                >
                                    {filter === "errors"
                                        ? "No rows with errors"
                                        : "No rows to display"}
                                </TableCell>
                            </TableRow>
                        ) : (
                            pageRows.map((row) => (
                                <TableRow
                                    key={row.row_number}
                                    className={cn(row.ui_meta.skipped && "opacity-40")}
                                >
                                    <TableCell className="text-muted-foreground text-xs">
                                        {row.row_number + 1}
                                    </TableCell>
                                    <TableCell
                                        className={cn(
                                            "font-medium",
                                            row.ui_meta.skipped && "line-through"
                                        )}
                                    >
                                        {row.processed_data.full_name || (
                                            <span className="text-destructive text-xs">
                                                Missing
                                            </span>
                                        )}
                                    </TableCell>
                                    <TableCell>
                                        {row.processed_data.gender || (
                                            <span className="text-destructive text-xs">—</span>
                                        )}
                                    </TableCell>
                                    <TableCell className="text-muted-foreground text-xs">
                                        {row.processed_data.date_of_birth ?? (
                                            <span className="text-muted-foreground/50">—</span>
                                        )}
                                    </TableCell>
                                    <TableCell>
                                        {row.ui_meta.errors.invalid_class ? (
                                            <ClassSelector
                                                classes={availableClasses}
                                                value={row.processed_data.class_id ?? ""}
                                                onChange={(classId) =>
                                                    handleClassChange(row.row_number, classId)
                                                }
                                                placeholder={row.ui_meta.errors.invalid_class}
                                            />
                                        ) : (
                                            <span className="text-sm">
                                                {row.processed_data.grade_level
                                                    ? `${row.processed_data.grade_level} ${row.processed_data.stream_name}`
                                                    : "—"}
                                            </span>
                                        )}
                                    </TableCell>
                                    <TableCell className="font-mono text-xs">
                                        {row.processed_data.nemis_number ?? "—"}
                                    </TableCell>
                                    <TableCell className="font-mono text-xs">
                                        {row.processed_data.assessment_number ?? "—"}
                                    </TableCell>
                                    <TableCell>
                                        <div className="flex flex-wrap gap-1">
                                            {row.ui_meta.errors.missing_required && (
                                                <Badge variant="destructive" className="text-xs">
                                                    Required
                                                </Badge>
                                            )}
                                            {row.ui_meta.errors.invalid_class && (
                                                <Badge variant="secondary" className="text-xs">
                                                    Class
                                                </Badge>
                                            )}
                                            {row.ui_meta.errors.invalid_date && (
                                                <Badge variant="secondary" className="text-xs">
                                                    Date
                                                </Badge>
                                            )}
                                            {renderErrorBadge(row)}
                                        </div>
                                    </TableCell>
                                    <TableCell>
                                        {row.ui_meta.has_error && (
                                            <Button
                                                variant="ghost"
                                                size="icon-sm"
                                                onClick={() =>
                                                    toggleSkip(row.row_number, row.ui_meta.skipped)
                                                }
                                            >
                                                {row.ui_meta.skipped ? (
                                                    <Undo2 className="size-4" />
                                                ) : (
                                                    <SkipForward className="size-4" />
                                                )}
                                            </Button>
                                        )}
                                    </TableCell>
                                </TableRow>
                            ))
                        )}
                    </TableBody>
                </Table>
            </div>

            {/* Pagination */}
            {totalPages > 1 && (
                <div className="flex items-center justify-between">
                    <p className="text-muted-foreground text-xs">
                        Page {currentPage} of {totalPages}
                    </p>
                    <div className="flex items-center gap-2">
                        <Button
                            variant="outline"
                            size="sm"
                            disabled={currentPage <= 1}
                            onClick={() => setCurrentPage((p) => Math.max(1, p - 1))}
                        >
                            Previous
                        </Button>
                        <Button
                            variant="outline"
                            size="sm"
                            disabled={currentPage >= totalPages}
                            onClick={() => setCurrentPage((p) => p + 1)}
                        >
                            Next
                        </Button>
                    </div>
                </div>
            )}

            {/* Actions */}
            <div className="flex items-center justify-between border-t pt-4">
                <div>
                    <p className="text-muted-foreground text-xs">
                        {hasUnresolvedErrors
                            ? "Resolve all errors or skip rows to continue"
                            : "All rows ready for import"}
                    </p>
                </div>
                <div className="flex items-center gap-2">
                    <Button variant="ghost" onClick={onClose}>
                        Cancel
                    </Button>
                    <Button
                        variant="ghost"
                        onClick={() => {
                            store.setStage("MAPPING");
                            onStageChange("MAPPING");
                        }}
                    >
                        Back
                    </Button>
                    <Button onClick={handleContinue} disabled={hasUnresolvedErrors}>
                        Continue
                    </Button>
                </div>
            </div>
        </div>
    );
}

// ─── Error type label helper ──────────────────────────────────────────────

function getErrorTypeLabel(errorType: string): string {
    switch (errorType) {
        case "SCHEMA_VALIDATION":
            return "Invalid data";
        case "DATABASE_CONSTRAINT":
            return "Duplicate";
        case "BUSINESS_RULE_VIOLATION":
            return "Rule violation";
        default:
            return errorType;
    }
}
