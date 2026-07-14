/**
 * TeacherHistoryView — shows past periods the teacher has taught with
 * attendance marking status. Same-day edit allowed; after that, read-only.
 *
 * Fetches the teacher's enriched timetable slots for the current academic year,
 * filters to past slots (before today), and shows their attendance marking status.
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
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { AttendanceEmptyState } from "./attendance-empty-state";

function getCurrentDayOfWeek(): number {
    const jsDay = new Date().getDay();
    return jsDay === 0 ? 7 : jsDay;
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

    const pastSlots = useMemo(() => {
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
            );
    }, [slotsData, currentDay]);

    if (isLoading) {
        return (
            <div className="space-y-4">
                <Skeleton className="h-8 w-64" />
                <Skeleton className="h-16 w-full rounded-lg" />
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
                <p className="text-muted-foreground text-sm">
                    Past periods you have taught. Same-day edits are allowed; older records are
                    read-only. Contact your admin for corrections after the same-day window closes.
                </p>
                <AttendanceEmptyState
                    icon={ClipboardList}
                    title="No past periods this week"
                    description="You don't have any completed periods for this week. History will populate once you've taught and marked attendance."
                >
                    <Button variant="outline" size="sm" asChild>
                        <Link href="/attendance">Return to attendance</Link>
                    </Button>
                </AttendanceEmptyState>
            </div>
        );
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

    return (
        <div className="space-y-6">
            <h1 className="text-2xl font-bold">Attendance History</h1>
            <p className="text-muted-foreground text-sm">
                Past periods you have taught. Same-day edits are allowed; older records are
                read-only. Contact your admin for corrections after the same-day window closes.
            </p>

            <Table>
                <TableHeader>
                    <TableRow>
                        <TableHead>Day</TableHead>
                        <TableHead>Period</TableHead>
                        <TableHead>Class</TableHead>
                        <TableHead>Time</TableHead>
                        <TableHead>Status</TableHead>
                        <TableHead className="w-20" />
                    </TableRow>
                </TableHeader>
                <TableBody>
                    {pastSlots.map((slot) => {
                        const isToday = slot.day_of_week === currentDay;
                        return (
                            <TableRow key={slot.id}>
                                <TableCell>
                                    {dayNames[slot.day_of_week] ?? `Day ${slot.day_of_week}`}
                                    {isToday && (
                                        <Badge variant="outline" className="ml-2 text-xs">
                                            Today
                                        </Badge>
                                    )}
                                </TableCell>
                                <TableCell className="font-medium">{slot.period_name}</TableCell>
                                <TableCell>{slot.class_name}</TableCell>
                                <TableCell className="text-muted-foreground">
                                    {slot.start_time} &ndash; {slot.end_time}
                                </TableCell>
                                <TableCell>
                                    {isToday ? (
                                        <Badge
                                            variant="secondary"
                                            className="bg-amber-100 text-amber-800"
                                        >
                                            Mark now
                                        </Badge>
                                    ) : (
                                        <Badge variant="outline">Completed</Badge>
                                    )}
                                </TableCell>
                                <TableCell>
                                    <Button
                                        variant="ghost"
                                        size="sm"
                                        onClick={() =>
                                            router.push(
                                                `/attendance/register?slot_id=${slot.id}&date=${new Date().toISOString().split("T")[0]}`
                                            )
                                        }
                                    >
                                        View
                                    </Button>
                                </TableCell>
                            </TableRow>
                        );
                    })}
                </TableBody>
            </Table>
        </div>
    );
}
