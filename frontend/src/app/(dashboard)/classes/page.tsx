/**
 * Classes listing page.
 *
 * Uses the shared DataTable component with filters for grade level, stream,
 * and academic year. Stream options are fetched dynamically from the streams
 * endpoint. Academic year options are fetched from the academic years endpoint.
 *
 * Maps to GET /api/v1/classes.
 */

"use client";

import Link from "next/link";
import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { GraduationCap, Calendar, Split } from "lucide-react";

import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import type { FilterGroup } from "@/components/shared/data-table/types";
import { listClasses, type Class } from "@/lib/api/classes";
import { listStreams } from "@/lib/api/streams";
import { StreamPill } from "@/features/settings-school";
import { GradeLevelPill, getGradeLevelFilterSubmenu } from "@/features/grade-level";
import { listAcademicYears } from "@/lib/api/academic-terms";

// ─── Columns ──────────────────────────────────────────────────────────────

const columns: DataTableColumn<Class>[] = [
    {
        id: "display_label",
        header: "Class",
        cell: (row) => (
            <Link href={`/classes/${row.id}`} className="font-medium hover:underline">
                {row.display_label}
            </Link>
        ),
    },
    {
        id: "grade_level",
        header: "Grade Level",
        width: "120px",
        cell: (row) => <GradeLevelPill grade={row.grade_level} />,
    },
    {
        id: "stream_name",
        header: "Stream",
        width: "120px",
        cell: (row) => <StreamPill name={row.stream_name} color={row.stream_color} />,
    },
    {
        id: "student_count",
        header: "Students",
        width: "100px",
        align: "right",
        cell: (row) => (
            <span className="text-muted-foreground tabular-nums">{row.student_count ?? 0}</span>
        ),
    },
];

// ─── Page ─────────────────────────────────────────────────────────────────

export default function ClassesPage() {
    // ── Fetch dynamic filter options ────────────────────────────────
    const { data: streamsData } = useQuery({
        queryKey: ["streams"],
        queryFn: () => listStreams(),
        staleTime: 5 * 60 * 1000,
    });

    const { data: academicYearsData } = useQuery({
        queryKey: ["academic-years"],
        queryFn: () => listAcademicYears(),
        staleTime: 5 * 60 * 1000,
    });

    // ── Build filter groups dynamically ─────────────────────────────
    const filterGroups = useMemo<FilterGroup[]>(() => {
        // Start with grade level and stream
        const items: FilterGroup["items"] = [
            {
                id: "grade_level",
                label: "Grade",
                icon: GraduationCap,
                type: "sub_menu_multi",
                submenu: getGradeLevelFilterSubmenu(),
            },
        ];

        // Add stream filter if data is available
        const streams = streamsData?.items ?? [];
        if (streams.length > 0) {
            items.push({
                id: "stream_id",
                label: "Stream",
                icon: Split,
                type: "sub_menu_multi",
                submenu: streams.map((s) => ({
                    id: s.id,
                    label: s.name,
                    value: s.id,
                    dotColor: s.color || undefined,
                })),
            });
        }

        // Add academic year filter if data is available
        const years = academicYearsData?.items ?? [];
        if (years.length > 0) {
            items.push({
                id: "academic_year_id",
                label: "Academic Year",
                icon: Calendar,
                type: "sub_menu_single",
                submenu: years.map((ay) => ({
                    id: ay.id,
                    label: ay.name,
                    value: ay.id,
                })),
            });
        }

        return [
            {
                id: "class_filters",
                label: "Filter by",
                items,
            },
        ];
    }, [streamsData, academicYearsData]);

    return (
        <DataTable
            addHref="/classes/add"
            queryKey={["classes"]}
            queryFn={listClasses}
            columns={columns}
            getRowId={(row) => row.id}
            isSearchable
            searchPlaceholder="Search by class name..."
            filterGroups={filterGroups}
            emptyState="No classes yet."
            noResultsState="No classes match your search or filters."
        />
    );
}
