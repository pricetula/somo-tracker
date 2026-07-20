/**
 * SessionList — displays attendance sessions with ability to filter by class and date.
 *
 * TEACHER / SCHOOL_ADMIN: view and manage lesson sessions.
 */
"use client";

import { useState } from "react";
import { format } from "date-fns";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useAttendanceSessions } from "@/features/attendance/hooks/use-attendance";
import type { SessionWithEnrichedData } from "@/features/attendance/types";

// ─── Helpers ──────────────────────────────────────────────────────────────

const statusVariant = (status: string) => {
    switch (status) {
        case "SUBMITTED":
            return "default" as const;
        case "SKIPPED":
            return "secondary" as const;
        default:
            return "outline" as const;
    }
};

function formatDate(dateStr: string) {
    try {
        return format(new Date(dateStr + "T00:00:00"), "EEE, MMM d, yyyy");
    } catch {
        return dateStr;
    }
}

// ─── Component ────────────────────────────────────────────────────────────

export function SessionList() {
    const [classFilter, setClassFilter] = useState("");
    const [dateFilter, setDateFilter] = useState("");

    const { data, isLoading, isError } = useAttendanceSessions({
        ...(classFilter ? { class_id: classFilter } : {}),
        ...(dateFilter ? { date: dateFilter } : {}),
    });

    if (isLoading) {
        return (
            <div className="space-y-3">
                {Array.from({ length: 5 }).map((_, i) => (
                    <Skeleton key={i} className="h-12 w-full" />
                ))}
            </div>
        );
    }

    if (isError) {
        return (
            <div className="bg-destructive/10 text-destructive rounded-md p-4 text-sm">
                Failed to load attendance sessions. Please try again.
            </div>
        );
    }

    const items = data?.items ?? [];

    if (items.length === 0) {
        return (
            <div className="text-muted-foreground py-8 text-center text-sm">
                No attendance sessions found.
            </div>
        );
    }

    return (
        <div className="space-y-4">
            <div className="flex flex-wrap gap-2">
                <input
                    type="text"
                    placeholder="Filter by class ID..."
                    value={classFilter}
                    onChange={(e) => setClassFilter(e.target.value)}
                    className="border-input bg-background rounded-md border px-3 py-1 text-sm"
                />
                <input
                    type="date"
                    value={dateFilter}
                    onChange={(e) => setDateFilter(e.target.value)}
                    className="border-input bg-background rounded-md border px-3 py-1 text-sm"
                />
                {(classFilter || dateFilter) && (
                    <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => {
                            setClassFilter("");
                            setDateFilter("");
                        }}
                    >
                        Clear
                    </Button>
                )}
            </div>

            <div className="space-y-2">
                {items.map((session: SessionWithEnrichedData) => (
                    <div
                        key={session.id}
                        className="bg-muted/30 flex items-center justify-between rounded-md px-4 py-3"
                    >
                        <div className="flex flex-col gap-1">
                            <span className="text-foreground font-medium">
                                {session.class_name} — {session.period_name}
                            </span>
                            <span className="text-muted-foreground text-xs">
                                {formatDate(session.date)} · {session.start_time}–{session.end_time}
                                {session.teacher_name && ` · ${session.teacher_name}`}
                                {session.learning_area_name && ` · ${session.learning_area_name}`}
                            </span>
                        </div>
                        <Badge variant={statusVariant(session.status)}>{session.status}</Badge>
                    </div>
                ))}
            </div>
        </div>
    );
}
