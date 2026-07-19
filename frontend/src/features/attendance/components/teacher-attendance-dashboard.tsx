/**
 * TeacherAttendanceDashboard — teacher's personal attendance view.
 * Pure shadcn: shows today's timetable slots; no cards/borders/hardcoded colours.
 */

"use client";

import Link from "next/link";
import { useMemo } from "react";
import { useMe } from "@/hooks/use-auth";
import { useEnrichedSlotList } from "@/features/timetable-structure/hooks/use-timetable-structure";
import { useAcademicYears } from "@/features/academic-terms/hooks/use-academic-terms";
import { Pencil, School } from "lucide-react";

import { StaticTable } from "@/components/shared/static-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import { Skeleton } from "@/components/ui/skeleton";
import { AttendanceEmptyState } from "./attendance-empty-state";

interface TeacherSlotRow {
    id: string;
    period_name: string;
    class_name: string;
    learning_area: string;
    start_time: string;
    end_time: string;
}

function todayStr(): string {
    return new Date().toISOString().split("T")[0];
}

function getCurrentDayOfWeek(): number {
    const jsDay = new Date().getDay();
    return jsDay === 0 ? 7 : jsDay;
}

const columns: DataTableColumn<TeacherSlotRow>[] = [
    {
        id: "period_name",
        header: "Period",
        width: "140px",
        cell: (row) => <span className="text-muted-foreground">{row.period_name}</span>,
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
        id: "class_name",
        header: "Class",
        width: "minmax(160px, 1fr)",
        cell: (row) => (
            <Link
                href={`/classes/${row.id}`}
                className="text-foreground font-medium hover:underline"
            >
                {row.class_name}
            </Link>
        ),
    },
    {
        id: "learning_area",
        header: "Subject",
        width: "minmax(160px, 1fr)",
        cell: (row) => (
            <span className="text-muted-foreground">{row.learning_area || "\u2014"}</span>
        ),
    },
    {
        id: "actions",
        header: "Register",
        width: "100px",
        align: "center",
        cell: (row) => (
            <Link
                href={`/attendance/register/${row.id}?date=${todayStr()}`}
                title={`Register attendance for ${row.class_name} \u00b7 ${row.period_name}`}
            >
                <Pencil className="text-muted-foreground hover:text-foreground h-4 w-4" />
            </Link>
        ),
    },
];

export function TeacherAttendanceDashboard() {
    const { data: me, isLoading: meLoading } = useMe();
    const { data: academicYears } = useAcademicYears();
    const currentYear = useMemo(
        () => academicYears?.items?.find((y) => y.is_current),
        [academicYears]
    );
    const currentDay = getCurrentDayOfWeek();

    const teacherId = me?.user_id;

    const { data: slotsData, isLoading: slotsLoading } = useEnrichedSlotList(
        currentYear?.id ?? "",
        teacherId ? { mode: "teacher", id: teacherId } : undefined
    );

    const isLoading = meLoading || slotsLoading;

    const rows = useMemo<TeacherSlotRow[]>(() => {
        if (!slotsData?.items?.length) return [];
        return slotsData.items
            .filter((s) => s.day_of_week === currentDay && !s.is_break)
            .sort((a, b) => a.start_time.localeCompare(b.start_time))
            .map((s) => ({
                id: s.id,
                period_name: s.period_name,
                class_name: s.class_name,
                learning_area: s.learning_area_name ?? "",
                start_time: s.start_time,
                end_time: s.end_time,
            }));
    }, [slotsData, currentDay]);

    if (isLoading) {
        return (
            <div className="space-y-4">
                <Skeleton className="h-8 w-48" />
                {Array.from({ length: 4 }).map((_, i) => (
                    <Skeleton key={i} className="h-10 w-full" />
                ))}
            </div>
        );
    }

    if (!rows.length) {
        return (
            <AttendanceEmptyState
                icon={School}
                title="No classes scheduled today"
                description="You don't have any timetable slots assigned for today. Contact your school admin if this seems wrong."
            />
        );
    }

    return (
        <div className="space-y-6">
            <p className="text-foreground text-2xl font-bold">My Attendance</p>
            <p className="text-muted-foreground">
                {rows.length} period{rows.length !== 1 ? "s" : ""} today
            </p>
            <StaticTable
                columns={columns}
                data={rows}
                getRowId={(row) => row.id}
                height={rows.length * 52 + 50}
            />
        </div>
    );
}
