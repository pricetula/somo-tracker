/**
 * AdminAttendanceDashboard — school-wide attendance completion view.
 *
 * Uses the shared DataTable component with filters for education level,
 * grade level, class, and completion status.
 *
 * Maps to GET /api/v1/attendance/dashboard.
 */

"use client";

import Link from "next/link";
import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { GraduationCap, BookOpen, CheckCircle, Pencil } from "lucide-react";

import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import type { FilterGroup } from "@/components/shared/data-table/types";
import { Badge } from "@/components/ui/badge";
import { listAdminAttendances, type CompletionStatus } from "@/lib/api/attendance";
import { listClasses, type Class } from "@/lib/api/classes";
import { getEducationLevelFilterSubmenu } from "@/features/education-level";
import { getGradeLevelFilterSubmenu } from "@/features/grade-level";

// ─── Columns ──────────────────────────────────────────────────────────────

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
            <Link href={`/classes/${row.class_id}`} className="font-medium hover:underline">
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
                return <span className="text-muted-foreground italic">—</span>;
            }
            return row.learning_area_id ? (
                <Link
                    href={`/curriculum/learning-areas/${row.learning_area_id}`}
                    className="hover:underline"
                >
                    {row.learning_area}
                </Link>
            ) : (
                <span>{row.learning_area}</span>
            );
        },
    },
    {
        id: "status",
        header: "Status",
        width: "120px",
        cell: (row) => (
            <Badge
                variant={row.is_complete ? "default" : "secondary"}
                className={row.is_complete ? "" : "bg-amber-100 text-amber-800 hover:bg-amber-100"}
            >
                {row.is_complete ? "Complete" : "Incomplete"}
            </Badge>
        ),
    },
    {
        id: "actions",
        header: "",
        width: "60px",
        align: "center",
        cell: (row) => (
            <Link
                href={`/attendance/register?slot_id=${row.slot_id}&date=${todayStr()}`}
                className="text-muted-foreground hover:text-foreground inline-flex items-center justify-center transition-colors"
                title={`Register attendance for ${row.class_name} · ${row.period_name}`}
            >
                <Pencil className="h-4 w-4" />
            </Link>
        ),
    },
];

// ─── Page ─────────────────────────────────────────────────────────────────

export function AdminAttendanceDashboard() {
    // ── Fetch classes for the class filter dropdown ──────────────
    const { data: classesData } = useQuery({
        queryKey: ["classes", "all"],
        queryFn: () => listClasses({ limit: 200 }),
        staleTime: 5 * 60 * 1000,
    });

    // ── Build filter groups dynamically ──────────────────────────
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

        // Add class filter if data is available
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

        // Add completion status filter
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
                        <Badge
                            variant="secondary"
                            className="pointer-events-none bg-amber-100 text-amber-800"
                        >
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
        <DataTable
            queryKey={["attendance", "dashboard"]}
            queryFn={listAdminAttendances}
            columns={columns}
            getRowId={(row) => row.slot_id}
            filterGroups={filterGroups}
            emptyState={
                <div className="text-muted-foreground flex flex-col items-center gap-2 py-12">
                    <p className="text-sm">No attendance records for today.</p>
                    <p className="text-xs">
                        Records will appear here once timetable slots are created and attendance is
                        marked.
                    </p>
                </div>
            }
            noResultsState="No attendance records match your filters."
            pageSize={50}
            height={600}
        />
    );
}
