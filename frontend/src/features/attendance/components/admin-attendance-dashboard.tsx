/**
 * AdminAttendanceDashboard — school-wide attendance completion view.
 * Pure shadcn: no borders/cards, no hardcoded colours.
 */

"use client";

import Link from "next/link";
import { useMemo, useState, useCallback } from "react";
import { useQuery } from "@tanstack/react-query";
import { STALE_TIMES } from "@/lib/query-config";
import {
    CalendarDays,
    GraduationCap,
    BookOpen,
    CheckCircle,
    Pencil,
    RotateCcw,
    Loader2,
    ChevronLeft,
    ChevronRight,
} from "lucide-react";

import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import type { FilterGroup } from "@/components/shared/data-table/types";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { DatePicker } from "@/components/ui/date-picker";
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
    DialogTrigger,
} from "@/components/ui/dialog";
import { listAdminAttendances, type CompletionStatus } from "@/lib/api/attendance";
import { listClasses, type Class } from "@/lib/api/classes";
import { getEducationLevelFilterSubmenu } from "@/features/education-level";
import { getGradeLevelFilterSubmenu } from "@/features/grade-level";
import { useAcademicTerms } from "@/features/academic-terms/hooks/use-academic-terms";
import { useComputeAttendanceSummaries } from "../hooks/use-attendance";

function todayStr(): string {
    return new Date().toISOString().split("T")[0];
}

const columns: DataTableColumn<CompletionStatus>[] = [
    {
        id: "period_name",
        header: "Period",
        width: "140px",
        cell: (row) => <span className="text-muted-foreground">{row.period_name}</span>,
    },
    {
        id: "class_name",
        header: "Class",
        width: "minmax(160px, 1fr)",
        cell: (row) => (
            <Link
                href={`/classes/${row.class_id}`}
                className="text-foreground font-medium hover:underline"
            >
                {row.class_name}
            </Link>
        ),
    },
    {
        id: "learning_area",
        header: "Learning Area",
        width: "minmax(160px, 1fr)",
        cell: (row) => {
            if (!row.learning_area) {
                return <span className="text-muted-foreground italic">\u2014</span>;
            }
            return row.learning_area_id ? (
                <Link
                    href={`/curriculum/learning-areas/${row.learning_area_id}`}
                    className="text-foreground hover:underline"
                >
                    {row.learning_area}
                </Link>
            ) : (
                <span className="text-foreground">{row.learning_area}</span>
            );
        },
    },
    {
        id: "status",
        header: "Status",
        width: "100px",
        cell: (row) => (
            <Badge variant={row.is_complete ? "default" : "secondary"}>
                {row.is_complete ? "Complete" : "Incomplete"}
            </Badge>
        ),
    },
    {
        id: "actions",
        header: "Register",
        width: "100px",
        align: "center",
        cell: (row) => (
            <Link
                href={`/attendance/register/${row.slot_id}?date=${todayStr()}`}
                title={`Register attendance for ${row.class_name} \u00b7 ${row.period_name}`}
            >
                <Pencil className="text-muted-foreground hover:text-foreground h-4 w-4" />
            </Link>
        ),
    },
];

export function AdminAttendanceDashboard() {
    const [selectedDate, setSelectedDate] = useState(todayStr());
    const [computeDialogOpen, setComputeDialogOpen] = useState(false);

    const computeSummaries = useComputeAttendanceSummaries();
    const { data: termsData } = useAcademicTerms();
    const currentTerm = useMemo(
        () => termsData?.items?.find((t) => t.is_current) ?? null,
        [termsData]
    );

    const handlePrevDay = useCallback(() => {
        const d = new Date(selectedDate);
        d.setDate(d.getDate() - 1);
        setSelectedDate(d.toISOString().split("T")[0]);
    }, [selectedDate]);

    const handleNextDay = useCallback(() => {
        const d = new Date(selectedDate);
        d.setDate(d.getDate() + 1);
        setSelectedDate(d.toISOString().split("T")[0]);
    }, [selectedDate]);

    const handleToday = useCallback(() => {
        setSelectedDate(todayStr());
    }, []);

    const { data: classesData } = useQuery({
        queryKey: ["classes", "all"],
        queryFn: () => listClasses({ limit: 200 }),
        staleTime: STALE_TIMES.REFERENCE_DATA,
    });

    const filterGroups = useMemo<FilterGroup[]>(() => {
        const items: FilterGroup["items"] = [
            {
                id: "education_level",
                label: "Education Level",
                icon: GraduationCap,
                type: "sub_menu_multi",
                submenu: getEducationLevelFilterSubmenu(),
            },
            {
                id: "grade_level",
                label: "Grade",
                icon: BookOpen,
                type: "sub_menu_multi",
                submenu: getGradeLevelFilterSubmenu(),
            },
        ];

        const allClasses = classesData?.items ?? [];
        if (allClasses.length > 0) {
            items.push({
                id: "class_id",
                label: "Class",
                icon: GraduationCap,
                type: "sub_menu_single",
                submenu: allClasses.map((c: Class) => ({
                    id: c.id,
                    label: c.display_label,
                    value: c.id,
                })),
            });
        }

        items.push({
            id: "is_complete",
            label: "Status",
            icon: CheckCircle,
            type: "sub_menu_single",
            submenu: [
                {
                    id: "complete",
                    label: (
                        <Badge variant="default" className="pointer-events-none">
                            Complete
                        </Badge>
                    ),
                    value: "complete",
                },
                {
                    id: "incomplete",
                    label: (
                        <Badge variant="secondary" className="pointer-events-none">
                            Incomplete
                        </Badge>
                    ),
                    value: "incomplete",
                },
            ],
        });

        return [
            {
                id: "attendance_filters",
                label: "Filter by",
                items,
            },
        ];
    }, [classesData]);

    return (
        <div className="space-y-4">
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                    <div className="flex items-center gap-2">
                        <CalendarDays className="text-muted-foreground h-4 w-4" />
                        <Label className="text-foreground font-medium">Date</Label>
                    </div>
                    <div className="flex items-center gap-1">
                        <Button
                            variant="outline"
                            size="icon"
                            className="size-8"
                            onClick={handlePrevDay}
                            title="Previous day"
                        >
                            <ChevronLeft className="size-4" />
                        </Button>
                        <DatePicker value={selectedDate} onChange={setSelectedDate} />
                        <Button
                            variant="outline"
                            size="icon"
                            className="size-8"
                            onClick={handleNextDay}
                            title="Next day"
                        >
                            <ChevronRight className="size-4" />
                        </Button>
                        <Button variant="ghost" size="sm" onClick={handleToday} className="text-xs">
                            Today
                        </Button>
                    </div>
                </div>

                <Dialog open={computeDialogOpen} onOpenChange={setComputeDialogOpen}>
                    <DialogTrigger asChild>
                        <Button variant="outline" size="sm">
                            <RotateCcw className="mr-2 h-4 w-4" />
                            Recompute Summaries
                        </Button>
                    </DialogTrigger>
                    <DialogContent>
                        <DialogHeader>
                            <DialogTitle>Recompute attendance summaries</DialogTitle>
                            <DialogDescription>
                                This will recalculate attendance percentages for all students in the
                                current term{currentTerm ? ` (${currentTerm.name})` : ""}. This
                                operation may take a few seconds.
                            </DialogDescription>
                        </DialogHeader>
                        <DialogFooter>
                            <Button variant="ghost" onClick={() => setComputeDialogOpen(false)}>
                                Cancel
                            </Button>
                            <Button
                                onClick={() => {
                                    if (currentTerm) {
                                        computeSummaries.mutate(currentTerm.id, {
                                            onSettled: () => setComputeDialogOpen(false),
                                        });
                                    }
                                }}
                                disabled={!currentTerm || computeSummaries.isPending}
                            >
                                {computeSummaries.isPending && (
                                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                                )}
                                Compute Now
                            </Button>
                        </DialogFooter>
                    </DialogContent>
                </Dialog>
            </div>

            <DataTable
                queryKey={["attendance", "dashboard", selectedDate]}
                queryFn={(params) => listAdminAttendances({ ...params, date: selectedDate })}
                columns={columns}
                getRowId={(row) => row.slot_id}
                filterGroups={filterGroups}
                emptyState={
                    <div className="text-muted-foreground flex flex-col items-center gap-2 py-12">
                        <p>No attendance records for {selectedDate}.</p>
                        <p className="text-xs">
                            Records will appear here once timetable slots are created and attendance
                            is marked.
                        </p>
                    </div>
                }
                noResultsState="No attendance records match your filters."
                pageSize={50}
                height={600}
            />
        </div>
    );
}
