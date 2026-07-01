/**
 * Scoring Grid — matrix layout for recording learner rubric results.
 *
 * Rows = students in the class, Columns = performance indicators from blueprint.
 * Each cell is a select with rubric levels (EE/ME/AE/BE).
 * Supports keyboard navigation (Tab, arrow keys).
 * Batch save button persists all changed cells.
 */

"use client";

import * as React from "react";
import { useVirtualizer } from "@tanstack/react-virtual";

import { Button } from "@/components/ui/button";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Loader2, Save } from "lucide-react";

import { RUBRIC_LEVELS } from "../types";
import type { LinkedIndicator, LearnerRubricResult } from "../types";

// ─── Rubric level colors ──────────────────────────────────────────────────

const RUBRIC_COLORS: Record<string, string> = {
    EE: "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400",
    ME: "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400",
    AE: "bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400",
    BE: "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400",
};

// ─── Grid cell value ──────────────────────────────────────────────────────

interface CellValue {
    rubricLevel: string;
    rawScore?: string | null;
}

// ─── Props ─────────────────────────────────────────────────────────────────

interface ScoringGridProps {
    /** Students in the class (id + name) */
    students: { id: string; full_name: string }[];
    /** Indicators from the blueprint */
    indicators: LinkedIndicator[];
    /** Previously saved results (keyed by student_id + indicator_id) */
    savedResults: Map<string, LearnerRubricResult>;
    /** Loading state */
    isLoading: boolean;
    /** Called on batch save */
    onSave: (
        results: Array<{
            student_id: string;
            indicator_id: string;
            rubric_level: string;
            score_type: string;
            raw_score?: string | null;
        }>
    ) => Promise<void>;
    /** Whether a save is in progress */
    isSaving: boolean;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function ScoringGrid({
    students,
    indicators,
    savedResults,
    isLoading,
    onSave,
    isSaving,
}: ScoringGridProps) {
    // Grid state: map of "student_id:indicator_id" → CellValue
    const [grid, setGrid] = React.useState<Map<string, CellValue>>(() => {
        const initial = new Map<string, CellValue>();
        for (const s of students) {
            for (const ind of indicators) {
                const key = `${s.id}:${ind.id}`;
                const saved = savedResults.get(key);
                if (saved) {
                    initial.set(key, {
                        rubricLevel: saved.rubric_level,
                        rawScore: saved.raw_score,
                    });
                }
            }
        }
        return initial;
    });

    // Tracks cells that have been modified during this editing session
    const [dirtyCells, setDirtyCells] = React.useState<Set<string>>(new Set());
    const [saveError, setSaveError] = React.useState<string | null>(null);

    // Update grid when savedResults changes (data refresh)
    React.useEffect(() => {
        setGrid((prev) => {
            const next = new Map(prev);
            for (const s of students) {
                for (const ind of indicators) {
                    const key = `${s.id}:${ind.id}`;
                    const saved = savedResults.get(key);
                    if (saved && !dirtyCells.has(key)) {
                        next.set(key, {
                            rubricLevel: saved.rubric_level,
                            rawScore: saved.raw_score,
                        });
                    }
                }
            }
            return next;
        });
    }, [savedResults, students, indicators, dirtyCells]);

    const setCellValue = React.useCallback(
        (studentId: string, indicatorId: string, value: CellValue) => {
            const key = `${studentId}:${indicatorId}`;
            setGrid((prev) => {
                const next = new Map(prev);
                next.set(key, value);
                return next;
            });
            setDirtyCells((prev) => new Set(prev).add(key));
            setSaveError(null);
        },
        []
    );

    const handleSave = async () => {
        setSaveError(null);
        const results: Array<{
            student_id: string;
            indicator_id: string;
            rubric_level: string;
            score_type: string;
            raw_score?: string | null;
        }> = [];

        for (const key of dirtyCells) {
            const [studentId, indicatorId] = key.split(":");
            const cell = grid.get(key);
            if (!cell || !cell.rubricLevel) continue;
            results.push({
                student_id: studentId,
                indicator_id: indicatorId,
                rubric_level: cell.rubricLevel,
                score_type: "Rubric_Direct",
                raw_score: cell.rawScore ?? null,
            });
        }

        if (results.length === 0) {
            return;
        }

        try {
            await onSave(results);
            setDirtyCells(new Set());
        } catch (err) {
            setSaveError(err instanceof Error ? err.message : "Failed to save scores");
        }
    };

    // ─── Virtual scrolling ─────────────────────────────────────────────────
    const parentRef = React.useRef<HTMLDivElement>(null);

    // eslint-disable-next-line react-hooks/incompatible-library
    const virtualizer = useVirtualizer({
        count: students.length,
        getScrollElement: () => parentRef.current,
        estimateSize: () => 48,
        overscan: 10,
    });

    const indicatorWidth = 180;
    const studentNameWidth = 180;

    if (isLoading) {
        return (
            <div className="space-y-2">
                <Skeleton className="h-10 w-full" />
                {Array.from({ length: 5 }).map((_, i) => (
                    <Skeleton key={i} className="h-12 w-full" />
                ))}
            </div>
        );
    }

    if (students.length === 0) {
        return (
            <div className="flex items-center justify-center py-16">
                <div className="text-center">
                    <p className="text-muted-foreground text-sm font-medium">
                        No students in this class
                    </p>
                    <p className="text-muted-foreground mt-1 text-xs">
                        Add students to the class before scoring.
                    </p>
                </div>
            </div>
        );
    }

    if (indicators.length === 0) {
        return (
            <div className="flex items-center justify-center py-16">
                <div className="text-center">
                    <p className="text-muted-foreground text-sm font-medium">
                        No indicators linked to this blueprint
                    </p>
                    <p className="text-muted-foreground mt-1 text-xs">
                        Link performance indicators to the blueprint before scoring.
                    </p>
                </div>
            </div>
        );
    }

    const dirtyCount = dirtyCells.size;

    return (
        <div className="flex flex-col gap-4">
            {/* Save bar */}
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                    {dirtyCount > 0 && (
                        <Badge variant="secondary" className="text-xs">
                            {dirtyCount} unsaved change{dirtyCount !== 1 ? "s" : ""}
                        </Badge>
                    )}
                    {saveError && <span className="text-destructive text-xs">{saveError}</span>}
                </div>
                <Button onClick={handleSave} disabled={dirtyCount === 0 || isSaving} size="sm">
                    {isSaving ? (
                        <>
                            <Loader2 className="mr-1.5 size-4 animate-spin" />
                            Saving…
                        </>
                    ) : (
                        <>
                            <Save className="mr-1.5 size-4" />
                            Save Scores
                        </>
                    )}
                </Button>
            </div>

            {/* Scrollable grid */}
            <div
                ref={parentRef}
                className="border-border/40 overflow-auto rounded-md border"
                style={{ maxHeight: "70vh", contain: "layout paint" }}
                role="grid"
                aria-label="Scoring grid"
            >
                <div
                    style={{
                        minWidth: `${studentNameWidth + indicators.length * indicatorWidth}px`,
                    }}
                >
                    {/* Column headers */}
                    <div className="bg-muted/30 border-border/40 sticky top-0 z-10 flex border-b">
                        <div
                            className="text-muted-foreground flex h-10 items-center px-3 text-xs font-medium tracking-wider uppercase"
                            style={{ width: studentNameWidth, flexShrink: 0 }}
                        >
                            Student
                        </div>
                        {indicators.map((ind) => (
                            <div
                                key={ind.id}
                                className="text-muted-foreground flex h-10 items-center justify-center truncate px-2 text-center text-xs font-medium"
                                style={{ width: indicatorWidth, flexShrink: 0 }}
                                title={ind.description}
                            >
                                {ind.description.length > 20
                                    ? `${ind.description.slice(0, 18)}…`
                                    : ind.description}
                            </div>
                        ))}
                    </div>

                    {/* Virtualized rows */}
                    <div
                        style={{ height: `${virtualizer.getTotalSize()}px`, position: "relative" }}
                    >
                        {virtualizer.getVirtualItems().map((virtualRow) => {
                            const student = students[virtualRow.index];
                            return (
                                <div
                                    key={virtualRow.key}
                                    className="hover:bg-muted/30 group border-border/40 flex items-center border-b"
                                    style={{
                                        position: "absolute",
                                        top: 0,
                                        left: 0,
                                        width: "100%",
                                        height: `${virtualRow.size}px`,
                                        transform: `translateY(${virtualRow.start}px)`,
                                    }}
                                    role="row"
                                >
                                    {/* Student name cell (sticky left) */}
                                    <div
                                        className="bg-background/95 sticky left-0 z-[5] flex items-center truncate px-3 text-sm font-medium backdrop-blur-sm"
                                        style={{ width: studentNameWidth, flexShrink: 0 }}
                                    >
                                        {student.full_name}
                                    </div>

                                    {/* Indicator cells */}
                                    {indicators.map((ind) => {
                                        const key = `${student.id}:${ind.id}`;
                                        const cell = grid.get(key);
                                        const isDirty = dirtyCells.has(key);
                                        const level = cell?.rubricLevel ?? "";

                                        return (
                                            <div
                                                key={ind.id}
                                                className="flex items-center justify-center px-1"
                                                style={{ width: indicatorWidth, flexShrink: 0 }}
                                            >
                                                <Select
                                                    value={level}
                                                    onValueChange={(val) =>
                                                        setCellValue(student.id, ind.id, {
                                                            rubricLevel: val,
                                                            rawScore: cell?.rawScore ?? null,
                                                        })
                                                    }
                                                >
                                                    <SelectTrigger
                                                        className={`h-8 text-xs ${
                                                            level
                                                                ? (RUBRIC_COLORS[level] ??
                                                                  "bg-muted text-muted-foreground")
                                                                : "bg-muted/50 text-muted-foreground"
                                                        } ${isDirty ? "ring-2 ring-sky-500/50" : ""}`}
                                                    >
                                                        <SelectValue placeholder="—" />
                                                    </SelectTrigger>
                                                    <SelectContent>
                                                        {RUBRIC_LEVELS.map((rl) => (
                                                            <SelectItem key={rl} value={rl}>
                                                                <span
                                                                    className={
                                                                        RUBRIC_COLORS[rl] ??
                                                                        "text-foreground"
                                                                    }
                                                                >
                                                                    {rl}
                                                                </span>
                                                            </SelectItem>
                                                        ))}
                                                    </SelectContent>
                                                </Select>
                                            </div>
                                        );
                                    })}
                                </div>
                            );
                        })}
                    </div>
                </div>
            </div>

            {/* Legend */}
            <div className="text-muted-foreground flex items-center gap-4 text-xs">
                <span className="font-medium">Rubric Key:</span>
                {RUBRIC_LEVELS.map((rl) => (
                    <Badge key={rl} variant="secondary" className={RUBRIC_COLORS[rl]}>
                        {rl}
                    </Badge>
                ))}
            </div>
        </div>
    );
}
