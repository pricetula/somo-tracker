/**
 * AdminAttendanceDashboard — school-wide attendance completion overview.
 * Pure shadcn: no borders, no cards, flat layout.
 *
 * Shows all class periods for today with their marking status.
 * Admins can drill into each class to mark or edit attendance.
 */

"use client";

import { useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { Calendar, CheckCircle2, Clock, AlertCircle } from "lucide-react";

import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";

import { AttendanceEmptyState } from "./attendance-empty-state";
import { useAdminDashboard } from "../hooks/use-attendance";
import type { CompletionStatus } from "../types";

export function AdminAttendanceDashboard() {
    const router = useRouter();
    const today = useMemo(() => new Date().toISOString().split("T")[0], []);
    const [selectedDate, setSelectedDate] = useState(today);

    const { data, isLoading, isError, error } = useAdminDashboard(selectedDate);

    const columns: DataTableColumn<CompletionStatus>[] = useMemo(
        () => [
            {
                id: "class",
                header: "Class",
                cell: (row) => (
                    <span className="text-foreground font-medium">{row.class_name}</span>
                ),
            },
            {
                id: "period",
                header: "Period",
                width: "120px",
                cell: (row) => <span className="text-foreground">{row.period_name}</span>,
            },
            {
                id: "subject",
                header: "Subject",
                width: "minmax(120px, 1fr)",
                cell: (row) => <span className="text-muted-foreground">{row.learning_area}</span>,
            },
            {
                id: "progress",
                header: "Marked",
                width: "120px",
                cell: (row) => {
                    if (row.is_complete) {
                        return (
                            <span className="flex items-center gap-1.5 text-emerald-600">
                                <CheckCircle2 className="size-4" />
                                {row.marked_slots}/{row.total_slots}
                            </span>
                        );
                    }
                    return (
                        <span className="text-destructive flex items-center gap-1.5">
                            <AlertCircle className="size-4" />
                            {row.marked_slots}/{row.total_slots}
                        </span>
                    );
                },
            },
            {
                id: "status",
                header: "Status",
                width: "100px",
                cell: (row) =>
                    row.is_complete ? (
                        <Badge variant="outline" className="text-emerald-600">
                            Complete
                        </Badge>
                    ) : (
                        <Badge variant="outline" className="text-destructive">
                            Incomplete
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
                        variant="outline"
                        size="sm"
                        onClick={() =>
                            router.push(`/attendance/register/${row.slot_id}?date=${selectedDate}`)
                        }
                    >
                        {row.is_complete ? "View" : "Mark"}
                    </Button>
                ),
            },
        ],
        [router, selectedDate]
    );

    if (isLoading) {
        return (
            <div className="space-y-6">
                <Skeleton className="h-8 w-64" />
                <Skeleton className="h-10 w-48" />
                <Skeleton className="h-64 w-full" />
            </div>
        );
    }

    if (isError) {
        return (
            <div className="text-destructive bg-destructive/10 p-4">
                Failed to load attendance dashboard.{" "}
                {error instanceof Error ? error.message : "Please try again."}
            </div>
        );
    }

    const items = data?.items ?? [];

    // Summary counts
    const completeCount = items.filter((i) => i.is_complete).length;
    const incompleteCount = items.filter((i) => !i.is_complete).length;
    const totalStudents = items.reduce((sum, i) => sum + i.total_slots, 0);
    const markedStudents = items.reduce((sum, i) => sum + i.marked_slots, 0);

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <h1 className="text-foreground text-2xl font-bold">Attendance Dashboard</h1>
                <div className="flex items-center gap-2">
                    <Label htmlFor="dashboard-date" className="sr-only">
                        Date
                    </Label>
                    <Input
                        id="dashboard-date"
                        type="date"
                        value={selectedDate}
                        onChange={(e) => setSelectedDate(e.target.value)}
                        className="w-44"
                    />
                </div>
            </div>

            {/* Summary stats — no borders, just wording */}
            <div className="flex flex-wrap items-center gap-6">
                <span className="text-muted-foreground text-sm">
                    {items.length} period{items.length !== 1 ? "s" : ""} today
                </span>
                <span className="flex items-center gap-1.5 text-sm">
                    <CheckCircle2 className="size-4 text-emerald-600" />
                    <span className="text-emerald-600">{completeCount} complete</span>
                </span>
                <span className="flex items-center gap-1.5 text-sm">
                    <AlertCircle className="text-destructive size-4" />
                    <span className="text-destructive">{incompleteCount} incomplete</span>
                </span>
                <span className="text-muted-foreground text-sm">
                    {markedStudents}/{totalStudents} students marked
                </span>
            </div>

            {items.length === 0 ? (
                <AttendanceEmptyState
                    icon={Calendar}
                    title="No classes scheduled"
                    description={
                        selectedDate === today
                            ? "No classes are scheduled for today."
                            : `No class periods found for ${selectedDate}.`
                    }
                >
                    {selectedDate !== today && (
                        <Button variant="outline" size="sm" onClick={() => setSelectedDate(today)}>
                            <Clock className="mr-2 size-4" />
                            Back to today
                        </Button>
                    )}
                </AttendanceEmptyState>
            ) : (
                <DataTable
                    queryKey={["attendance", "dashboard", selectedDate]}
                    queryFn={() =>
                        Promise.resolve({
                            items,
                            total: items.length,
                        })
                    }
                    columns={columns}
                    getRowId={(row) => row.slot_id}
                    emptyState="No classes found for this date."
                    noResultsState="No periods match your filters."
                    pageSize={100}
                    height={Math.min(items.length * 44 + 60, 600)}
                />
            )}
        </div>
    );
}
