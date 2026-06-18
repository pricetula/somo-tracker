"use client";

import * as React from "react";
import { format, subDays } from "date-fns";
import { Filter, RotateCcw } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import {
    Select,
    SelectTrigger,
    SelectValue,
    SelectContent,
    SelectItem,
} from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

import {
    useCbcAttendancePeriodSummaries,
    useCbcAttendanceGaps,
} from "@/features/cbc/hooks/use-cbc-attendance";
import type { AttendancePeriodSummary, AttendanceGap } from "@/features/cbc/types";

// ─── Props ────────────────────────────────────────────────────────────────

interface CbcAttendancePeriodListProps {
    classId: string;
    learningAreaOptions: Array<{ id: string; name: string }>;
    /** Currently selected period ID (for highlighting). */
    selectedPeriodId: string | null;
    onSelectPeriod: (periodId: string) => void;
    /** Called when user wants to fill a gap. */
    onFillGap: (gap: AttendanceGap) => void;
    /** Called when a period needs a new attendance session started. */
    onStartNewPeriod: (learningAreaId: string, date: string) => void;
    academicTermId: string;
}

// ─── Status pill for a period row ─────────────────────────────────────────

function PeriodStatusPill({ summary }: { summary: AttendancePeriodSummary }) {
    const total = summary.total_students;
    const unmarked = summary.unmarked_count;
    const marked = total - unmarked;

    if (total === 0) {
        return <span className="text-muted-foreground text-[10px]">No students</span>;
    }

    // Calculate widths for the stacked bar
    const presentPct = (summary.present_count / total) * 100;
    const absentPct = (summary.absent_count / total) * 100;
    const latePct = (summary.late_count / total) * 100;
    const excusedPct = (summary.excused_count / total) * 100;
    const unmarkedPct = (unmarked / total) * 100;

    return (
        <div className="flex items-center gap-2">
            {/* Stacked bar */}
            <div className="flex h-2 w-16 overflow-hidden rounded-full bg-gray-100">
                {summary.present_count > 0 && (
                    <div
                        className="h-full bg-green-500 transition-all"
                        style={{ width: `${presentPct}%` }}
                    />
                )}
                {summary.absent_count > 0 && (
                    <div
                        className="h-full bg-red-500 transition-all"
                        style={{ width: `${absentPct}%` }}
                    />
                )}
                {summary.late_count > 0 && (
                    <div
                        className="h-full bg-amber-400 transition-all"
                        style={{ width: `${latePct}%` }}
                    />
                )}
                {summary.excused_count > 0 && (
                    <div
                        className="h-full bg-gray-400 transition-all"
                        style={{ width: `${excusedPct}%` }}
                    />
                )}
                {unmarked > 0 && (
                    <div
                        className="h-full bg-gray-200 transition-all"
                        style={{ width: `${unmarkedPct}%` }}
                    />
                )}
            </div>

            {/* Counts text */}
            <span className="text-muted-foreground text-[10px] whitespace-nowrap">
                {marked}/{total}
            </span>
        </div>
    );
}

// ─── Gap row ──────────────────────────────────────────────────────────────

function GapRow({ gap, onFill }: { gap: AttendanceGap; onFill: () => void }) {
    return (
        <div className="flex items-center gap-3 border-b border-dashed border-red-200 bg-red-50/50 px-3 py-2">
            <div className="flex size-6 shrink-0 items-center justify-center rounded-full bg-red-100">
                <span className="text-[10px] font-bold text-red-600">!</span>
            </div>
            <div className="min-w-0 flex-1">
                <p className="truncate text-xs font-medium text-red-700">
                    {gap.learning_area_name}
                </p>
                <p className="text-[10px] text-red-500">
                    {gap.start_time}–{gap.end_time} · No attendance taken
                </p>
            </div>
            <Button
                variant="outline"
                size="sm"
                className="h-7 shrink-0 border-red-300 text-[10px] text-red-600 hover:bg-red-100"
                onClick={onFill}
            >
                Take now
            </Button>
        </div>
    );
}

// ─── Component ────────────────────────────────────────────────────────────

export function CbcAttendancePeriodList({
    classId,
    learningAreaOptions,
    selectedPeriodId,
    onSelectPeriod,
    onFillGap,
}: CbcAttendancePeriodListProps) {
    // ── Date range state ─────────────────────────────────────────────
    const [dateFrom, setDateFrom] = React.useState(format(subDays(new Date(), 30), "yyyy-MM-dd"));
    const [dateTo, setDateTo] = React.useState(format(new Date(), "yyyy-MM-dd"));
    const [filterLearningArea, setFilterLearningArea] = React.useState<string>("all");
    const [showGapsOnly, setShowGapsOnly] = React.useState(false);

    // ── Data ─────────────────────────────────────────────────────────
    const { data: summaries = [], isLoading: summariesLoading } = useCbcAttendancePeriodSummaries(
        classId,
        dateFrom,
        dateTo
    );

    const { data: gaps = [], isLoading: gapsLoading } = useCbcAttendanceGaps(
        classId,
        dateFrom,
        dateTo
    );

    // ── Filtered summaries ───────────────────────────────────────────
    const filteredSummaries = React.useMemo(() => {
        let result = summaries;
        if (filterLearningArea !== "all") {
            result = result.filter((s) => s.cbc_learning_area_id === filterLearningArea);
        }
        // Sort by date descending, most recent first
        return [...result].sort(
            (a, b) => new Date(b.date_recorded).getTime() - new Date(a.date_recorded).getTime()
        );
    }, [summaries, filterLearningArea]);

    // ── Quick date range presets ─────────────────────────────────────
    const setRangeThisWeek = () => {
        const now = new Date();
        const day = now.getDay();
        const monOffset = day === 0 ? 6 : day - 1;
        const mon = subDays(now, monOffset);
        setDateFrom(format(mon, "yyyy-MM-dd"));
        setDateTo(format(now, "yyyy-MM-dd"));
    };

    const setRangeThisMonth = () => {
        const now = new Date();
        setDateFrom(format(new Date(now.getFullYear(), now.getMonth(), 1), "yyyy-MM-dd"));
        setDateTo(format(now, "yyyy-MM-dd"));
    };

    const setRangeLast30 = () => {
        const now = new Date();
        setDateFrom(format(subDays(now, 30), "yyyy-MM-dd"));
        setDateTo(format(now, "yyyy-MM-dd"));
    };

    const setRangeThisTerm = () => {
        // Reset to term boundaries (provided by parent as a hint)
        setDateFrom(format(subDays(new Date(), 90), "yyyy-MM-dd"));
        setDateTo(format(new Date(), "yyyy-MM-dd"));
    };

    // ── Loading ──────────────────────────────────────────────────────
    if (summariesLoading || gapsLoading) {
        return (
            <div className="space-y-2 p-3">
                <Skeleton className="h-4 w-32" />
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-8 w-full" />
                <Skeleton className="h-8 w-full" />
            </div>
        );
    }

    return (
        <div className="space-y-3">
            {/* ── Filters bar ────────────────────────────────────────── */}
            <div className="flex flex-wrap items-center gap-2">
                {/* Learning area filter */}
                <div className="w-44">
                    <Select value={filterLearningArea} onValueChange={setFilterLearningArea}>
                        <SelectTrigger className="h-7 text-xs">
                            <SelectValue placeholder="All learning areas" />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem value="all">All learning areas</SelectItem>
                            {learningAreaOptions.map((la) => (
                                <SelectItem key={la.id} value={la.id}>
                                    {la.name}
                                </SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                </div>

                {/* Date range presets */}
                <div className="flex items-center gap-1">
                    <Button
                        variant="ghost"
                        size="sm"
                        className="h-7 px-2 text-[10px]"
                        onClick={setRangeThisWeek}
                    >
                        This week
                    </Button>
                    <Button
                        variant="ghost"
                        size="sm"
                        className="h-7 px-2 text-[10px]"
                        onClick={setRangeThisMonth}
                    >
                        This month
                    </Button>
                    <Button
                        variant="ghost"
                        size="sm"
                        className="h-7 px-2 text-[10px]"
                        onClick={setRangeLast30}
                    >
                        Last 30d
                    </Button>
                    <Button
                        variant="ghost"
                        size="sm"
                        className="h-7 px-2 text-[10px]"
                        onClick={setRangeThisTerm}
                    >
                        Term
                    </Button>
                </div>

                {/* Gaps-only toggle */}
                <Button
                    variant={showGapsOnly ? "secondary" : "outline"}
                    size="sm"
                    className={cn(
                        "h-7 px-2 text-[10px]",
                        showGapsOnly && "border-red-300 bg-red-50 text-red-700"
                    )}
                    onClick={() => setShowGapsOnly((prev) => !prev)}
                >
                    <Filter className="mr-1 size-2.5" />
                    Gaps only
                    {gaps.length > 0 && (
                        <Badge variant="secondary" className="ml-1 h-4 px-1 text-[8px]">
                            {gaps.length}
                        </Badge>
                    )}
                </Button>

                {(showGapsOnly || filterLearningArea !== "all") && (
                    <Button
                        variant="ghost"
                        size="sm"
                        className="h-7 px-2 text-[10px] text-gray-400"
                        onClick={() => {
                            setShowGapsOnly(false);
                            setFilterLearningArea("all");
                        }}
                    >
                        <RotateCcw className="mr-1 size-2.5" />
                        Reset
                    </Button>
                )}
            </div>

            {/* ── Content ────────────────────────────────────────────── */}
            <div className="max-h-96 overflow-y-auto rounded-md border">
                {/* Gaps section (shown when gaps-only toggle is off, but there ARE gaps) */}
                {!showGapsOnly && gaps.length > 0 && (
                    <div className="border-b border-red-200 bg-red-50/30 px-3 py-1.5">
                        <p className="text-[10px] font-medium text-red-600">
                            {gaps.length} unattended slot{gaps.length !== 1 ? "s" : ""} in this
                            period
                        </p>
                    </div>
                )}

                {/* Gaps-only view */}
                {showGapsOnly && (
                    <div>
                        {gaps.length === 0 ? (
                            <div className="flex items-center justify-center py-6">
                                <p className="text-muted-foreground text-xs">
                                    No gaps found — all scheduled slots have attendance recorded.
                                </p>
                            </div>
                        ) : (
                            gaps.map((gap) => (
                                <GapRow key={gap.slot_id} gap={gap} onFill={() => onFillGap(gap)} />
                            ))
                        )}
                    </div>
                )}

                {/* Regular period list (hidden in gaps-only view) */}
                {!showGapsOnly && (
                    <div>
                        {filteredSummaries.length === 0 ? (
                            <div className="flex items-center justify-center py-6">
                                <p className="text-muted-foreground text-xs">
                                    No attendance periods found for the selected filters.
                                </p>
                            </div>
                        ) : (
                            filteredSummaries.map((summary) => {
                                const isSelected = summary.id === selectedPeriodId;
                                const isComplete = summary.unmarked_count === 0;
                                return (
                                    <button
                                        key={summary.id}
                                        type="button"
                                        onClick={() => onSelectPeriod(summary.id)}
                                        className={cn(
                                            "flex w-full items-center gap-3 border-b px-3 py-2.5 text-left transition-colors hover:bg-gray-50",
                                            isSelected && "bg-teal-50 hover:bg-teal-50"
                                        )}
                                    >
                                        {/* Date */}
                                        <div className="min-w-0 shrink-0">
                                            <p className="text-xs font-medium">
                                                {format(new Date(summary.date_recorded), "MMM d")}
                                            </p>
                                            <p className="text-muted-foreground text-[10px]">
                                                {format(new Date(summary.date_recorded), "EEE")}
                                            </p>
                                        </div>

                                        {/* Vertical divider */}
                                        <div className="bg-border h-8 w-px" />

                                        {/* Learning area & recorder */}
                                        <div className="min-w-0 flex-1">
                                            <p className="truncate text-xs font-medium">
                                                {summary.learning_area_name}
                                            </p>
                                            <p className="text-muted-foreground truncate text-[10px]">
                                                by {summary.recorded_by_name}
                                            </p>
                                        </div>

                                        {/* Completion status indicator */}
                                        <div className="flex items-center gap-2">
                                            <PeriodStatusPill summary={summary} />
                                            {isComplete ? (
                                                <span className="inline-flex items-center gap-0.5 text-[10px] font-medium text-green-600">
                                                    <span className="inline-block size-1.5 rounded-full bg-green-500" />
                                                    Complete
                                                </span>
                                            ) : (
                                                <span className="inline-flex items-center gap-0.5 text-[10px] font-medium text-amber-600">
                                                    <span className="inline-block size-1.5 rounded-full bg-amber-400" />
                                                    {summary.unmarked_count} unmarked
                                                </span>
                                            )}
                                        </div>
                                    </button>
                                );
                            })
                        )}
                    </div>
                )}
            </div>

            {/* Summary footer */}
            {!showGapsOnly && filteredSummaries.length > 0 && (
                <p className="text-muted-foreground text-[10px]">
                    {filteredSummaries.length} period{filteredSummaries.length !== 1 ? "s" : ""} ·{" "}
                    {filteredSummaries.reduce((sum, s) => sum + s.total_students, 0)} total marks
                </p>
            )}
        </div>
    );
}
