"use client";

import React from "react";
import { format } from "date-fns";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Skeleton } from "@/components/ui/skeleton";
import { useSchoolAttendanceKPIs } from "@/features/attendance/hooks/use-school-attendance-kpis";
import { getErrorMessage } from "@/lib/errors";

/**
 * School Attendance Command Center — macro-level attendance KPIs for the
 * School Administrator dashboard.
 *
 * Renders four headline metrics fetched from
 * GET /api/v1/attendance/kpis/school: today's attendance rate, active term
 * rate, unmarked timetable slots, and skipped (cancelled) sessions.
 */
export function SchoolAttendanceKPIs() {
    // KPIs are anchored to "today" — the backend derives the active term
    // from this date when no term_id is passed.
    const today = React.useMemo(() => format(new Date(), "yyyy-MM-dd"), []);
    const { data, isLoading, isError, error } = useSchoolAttendanceKPIs(today);

    const heading = <h2 className="text-foreground font-medium">Attendance summary</h2>;

    if (isLoading) {
        return (
            <section className="space-y-4">
                {heading}
                <div className="grid grid-cols-2 gap-6 lg:grid-cols-4">
                    {Array.from({ length: 4 }).map((_, index) => (
                        <Skeleton key={index} className="h-16 w-full" />
                    ))}
                </div>
            </section>
        );
    }

    if (isError) {
        return (
            <section className="space-y-4">
                {heading}
                <Alert variant="destructive">
                    <AlertTitle>Unable to load attendance</AlertTitle>
                    <AlertDescription>{getErrorMessage(error)}</AlertDescription>
                </Alert>
            </section>
        );
    }

    // Data is defined once the query is neither loading nor erroring; the
    // guard protects against the brief refetch window where data is retained
    // but undefined on the very first render.
    if (!data) return null;

    const stats = [
        {
            label: "Today's attendance",
            value: `${data.todays_attendance_rate.toFixed(1)}%`,
            hint: `${data.total_present} present of ${data.total_marked_records} marked`,
            hintClass: "text-muted-foreground",
        },
        {
            label: "Active term rate",
            value: `${data.active_term_attendance_rate.toFixed(2)}%`,
            hint: "School average this term",
            hintClass: "text-muted-foreground",
        },
        {
            label: "Unmarked slots",
            value: String(data.unmarked_slots_today),
            hint: data.unmarked_slots_today > 0 ? "Action required" : "All slots marked",
            hintClass: data.unmarked_slots_today > 0 ? "text-destructive" : "text-muted-foreground",
        },
        {
            label: "Skipped sessions",
            value: String(data.skipped_sessions_today),
            hint: "Cancelled lessons",
            hintClass: "text-muted-foreground",
        },
    ];

    return (
        <section className="mb-10 max-w-sm space-y-4 border-r">
            {heading}
            <div className="grid grid-cols-2 gap-6">
                {stats.map((stat) => (
                    <div key={stat.label} className="flex flex-col gap-1">
                        <p className="text-foreground font-semibold">{stat.value}</p>
                        <p className="text-muted-foreground">{stat.label}</p>
                        <p className={`${stat.hintClass}`}>{stat.hint}</p>
                    </div>
                ))}
            </div>
        </section>
    );
}
