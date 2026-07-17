/**
 * TeacherHistoryView — shows past periods the teacher has taught with
 * attendance marking status using the shared DataTable component.
 */

"use client";

import { useMemo } from "react";
import { useRouter } from "next/navigation";
import { useMe } from "@/hooks/use-auth";
import { useEnrichedSlotList } from "@/features/timetable-structure/hooks/use-timetable-structure";
import { useAcademicYears } from "@/features/academic-terms/hooks/use-academic-terms";
import Link from "next/link";
import { AlertCircle, ClipboardList } from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import { AttendanceEmptyState } from "./attendance-empty-state";

function getCurrentDayOfWeek(): number {
    const jsDay = new Date().getDay();
    return jsDay === 0 ? 7 : jsDay;
}

function dateForDayOfWeek(dayOfWeek: number): string {
    const today = new Date();
    const currentDow = getCurrentDayOfWeek();
    const diff = dayOfWeek - currentDow;
    const target = new Date(today);
    target.setDate(today.getDate() + diff);
    return target.toISOString().split("T")[0];
}

const dayNames: Record<number, string> = {
    1: "Monday",
    2: "Tuesday",
    3: "Wednesday",
    4: "Thursday",
    5: "Friday",
    6: "Saturday",
    7: "Sunday",
};

interface PastSlotRow {
    id: string;
    day_of_week: number;
    slot_date: string;
    period_name: string;
    class_name: string;
    start_time: string;
    end_time: string;
    is_today: boolean;
}

export function TeacherHistoryView() {
    const router = useRouter();
    const { data: me, isLoading: meLoading } = useMe();
    const { data: academicYears, isLoading: yearsLoading } = useAcademicYears();
    const currentYear = useMemo(
        () => academicYears?.items?.find((y) => y.is_current),
        [academicYears]
    );
    const currentDay = getCurrentDayOfWeek();

    const enabled = !!currentYear?.id && !!me?.user_id;

    const { data: slotsData, isLoading: slotsLoading } = useEnrichedSlotList(
        currentYear?.id ?? "",
        enabled ? { mode: "teacher", id: me!.user_id } : undefined
    );

    const isLoading = meLoading || yearsLoading || slotsLoading;

    const pastSlots = useMemo<PastSlotRow[]>(() => {
        if (!slotsData?.items) return [];
        return slotsData.items
            .filter(
                (s) =>
                    !s.is_break &&
                    (s.day_of_week < currentDay ||
                        (s.day_of_week === currentDay &&
                            s.end_time <=
                                new Date().toLocaleTimeString("en-GB", {
                                    hour: "2-digit",
                                    minute: "2-digit",
                                })))
            )
            .sort(
                (a, b) => b.day_of_week - a.day_of_week || b.start_time.localeCompare(a.start_time)
            )
            .map((s) => ({
                id: s.id,
                day_of_week: s.day_of_week,
                slot_date: dateForDayOfWeek(s.day_of_week),
                period_name: s.period_name,
                class_name: s.class_name,
                start_time: s.start_time,
                end_time: s.end_time,
                is_today: s.day_of_week === currentDay,
            }));
    }, [slotsData, currentDay]);

    // ── Columns ───────────────────────────────────────────────────────
    const columns: DataTableColumn<PastSlotRow>[] = [
        {
            id: "day",
            header: "Day",
            cell: (row) => (
                <div className="flex items-center gap-2">
                    <span>{dayNames[row.day_of_week] ?? `Day ${row.day_of_week}`}</span>
                    <span className="text-muted-foreground text-xs">{row.slot_date}</span>
                    {row.is_today && (
                        <Badge variant="outline" className="text-xs">
                            Today
                        </Badge>
                    )}
                </div>
            ),
        },
        {
            id: "period",
            header: "Period",
            width: "120px",
            cell: (row) => <span className="font-medium">{row.period_name}</span>,
        },
        {
            id: "class",
            header: "Class",
            width: "minmax(120px, 1fr)",
            cell: (row) => <span>{row.class_name}</span>,
        },
        {
            id: "time",
            header: "Time",
            width: "140px",
            cell: (row) => (
                <span className="text-muted-foreground">
                    {row.start_time} &ndash; {row.end_time}
                </span>
            ),
        },
        {
            id: "status",
            header: "Status",
            width: "120px",
            cell: (row) => (
                <Badge
                    variant={row.is_today ? "secondary" : "outline"}
                    className={row.is_today ? "bg-amber-100 text-amber-800" : ""}
                >
                    {row.is_today ? "Mark now / View" : "Completed"}
                </Badge>
            ),
        },
        {
            id: "actions",
            header: "",
            width: "80px",
            align: "right",
            cell: (row) => (
                <Button
                    variant="ghost"
                    size="sm"
                    onClick={() =>
                        router.push(`/attendance/register/${row.id}?date=${row.slot_date}`)
                    }
                >
                    View
                </Button>
            ),
        },
    ];

    // ── Loading ───────────────────────────────────────────────────────
    if (isLoading) {
        return (
            <div className="space-y-4">
                <Skeleton className="h-8 w-64" />
                <Skeleton className="h-16 w-full rounded-lg" />
                <Skeleton className="h-16 w-full rounded-lg" />
            </div>
        );
    }

    if (!me) {
        return (
            <div className="border-destructive/50 text-destructive rounded-md border p-4">
                Unable to verify your session. Please log in again.
            </div>
        );
    }

    if (!currentYear) {
        return (
            <AttendanceEmptyState
                icon={AlertCircle}
                title="No active academic year"
                description="An academic year must be set as current before attendance history can be viewed."
            >
                <Button variant="outline" size="sm" asChild>
                    <Link href="/settings">Contact school admin</Link>
                </Button>
            </AttendanceEmptyState>
        );
    }

    if (pastSlots.length === 0) {
        return (
            <div className="space-y-6">
                <h1 className="text-2xl font-bold">Attendance History</h1>
                <p className="text-muted-foreground">
                    Past periods you have taught. Same-day edits are allowed; older records are
                    read-only.
                </p>
                <AttendanceEmptyState
                    icon={ClipboardList}
                    title="No past periods this week"
                    description="You don't have any completed periods for this week."
                >
                    <Button variant="outline" size="sm" asChild>
                        <Link href="/attendance">Return to attendance</Link>
                    </Button>
                </AttendanceEmptyState>
            </div>
        );
    }

    const queryFn = () => Promise.resolve({ items: pastSlots, total: pastSlots.length });

    return (
        <div className="space-y-6">
            <h1 className="text-2xl font-bold">Attendance History</h1>
            <p className="text-muted-foreground">
                Past periods you have taught. Same-day edits are allowed; older records are
                read-only. Contact your admin for corrections after the same-day window closes.
            </p>

            <DataTable
                queryKey={["teacher-history", me?.user_id, currentYear?.id]}
                queryFn={queryFn}
                columns={columns}
                getRowId={(row) => row.id}
                emptyState="No past periods this week."
                noResultsState="No periods match your search."
                pageSize={50}
                height={Math.min(pastSlots.length * 44 + 60, 500)}
            />
        </div>
    );
}
