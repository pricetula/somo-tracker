/**
 * SummaryTable — displays term-level attendance summaries for a class.
 *
 * SCHOOL_ADMIN / TEACHER: view attendance percentages per student and learning area.
 */
"use client";

import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import {
    useClassTermSummary,
    useRefreshSummaries,
} from "@/features/attendance/hooks/use-attendance";

// ─── Helpers ──────────────────────────────────────────────────────────────

function percentageColor(pct: number) {
    if (pct >= 90) return "text-emerald-600";
    if (pct >= 75) return "text-amber-600";
    return "text-destructive";
}

// ─── Component ────────────────────────────────────────────────────────────

interface SummaryTableProps {
    classId: string;
    termId?: string;
}

export function SummaryTable({ classId, termId }: SummaryTableProps) {
    const { data: summaries, isLoading, isError } = useClassTermSummary(classId, termId);
    const refresh = useRefreshSummaries();

    if (isLoading) {
        return (
            <div className="space-y-3">
                {Array.from({ length: 5 }).map((_, i) => (
                    <Skeleton key={i} className="h-10 w-full" />
                ))}
            </div>
        );
    }

    if (isError) {
        return (
            <div className="bg-destructive/10 text-destructive rounded-md p-4 text-sm">
                Failed to load attendance summaries. Please try again.
            </div>
        );
    }

    const items = summaries ?? [];

    if (items.length === 0) {
        return (
            <div className="text-muted-foreground py-8 text-center text-sm">
                No attendance summaries available for this class and term.
            </div>
        );
    }

    return (
        <div className="space-y-4">
            <div className="flex items-center justify-between">
                <p className="text-muted-foreground text-sm">
                    {items.length} summary {items.length === 1 ? "record" : "records"}
                </p>
                {termId && (
                    <Button
                        variant="outline"
                        size="sm"
                        onClick={() => refresh.mutate(termId)}
                        disabled={refresh.isPending}
                    >
                        {refresh.isPending ? "Refreshing..." : "Refresh"}
                    </Button>
                )}
            </div>

            <div className="overflow-x-auto">
                <table className="w-full text-sm">
                    <thead>
                        <tr className="text-muted-foreground border-b text-xs uppercase">
                            <th className="px-3 py-2 text-left font-medium">Student</th>
                            <th className="px-3 py-2 text-left font-medium">Learning Area</th>
                            <th className="px-3 py-2 text-center font-medium">Present</th>
                            <th className="px-3 py-2 text-center font-medium">Absent</th>
                            <th className="px-3 py-2 text-center font-medium">Late</th>
                            <th className="px-3 py-2 text-center font-medium">Excused</th>
                            <th className="px-3 py-2 text-right font-medium">%</th>
                        </tr>
                    </thead>
                    <tbody>
                        {items.map((summary) => (
                            <tr key={summary.id} className="border-b last:border-0">
                                <td className="text-foreground px-3 py-2 font-medium">
                                    {summary.student_id}
                                </td>
                                <td className="text-muted-foreground px-3 py-2">
                                    {summary.learning_area_name ?? "—"}
                                </td>
                                <td className="px-3 py-2 text-center">{summary.periods_present}</td>
                                <td className="px-3 py-2 text-center">{summary.periods_absent}</td>
                                <td className="px-3 py-2 text-center">{summary.periods_late}</td>
                                <td className="px-3 py-2 text-center">{summary.periods_excused}</td>
                                <td
                                    className={`px-3 py-2 text-right font-medium ${percentageColor(summary.attendance_percentage)}`}
                                >
                                    {summary.attendance_percentage.toFixed(1)}%
                                </td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>
        </div>
    );
}
