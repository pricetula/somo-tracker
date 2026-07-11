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
import { GraduationCap, BookOpen, Calendar } from "lucide-react";

import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import type { FilterGroup } from "@/components/shared/data-table/types";
import { listClasses, type Class } from "@/lib/api/classes";
import { listStreams } from "@/lib/api/streams";
import { listAcademicYears } from "@/lib/api/academic-terms";

// ─── Grade Level Submenu (single-select) ─────────────────────────────────

const GRADE_LEVEL_SUBMENU = [
    { id: "pp1", label: "PP1", value: "PP1" },
    { id: "pp2", label: "PP2", value: "PP2" },
    { id: "g1", label: "Grade 1", value: "G1" },
    { id: "g2", label: "Grade 2", value: "G2" },
    { id: "g3", label: "Grade 3", value: "G3" },
    { id: "g4", label: "Grade 4", value: "G4" },
    { id: "g5", label: "Grade 5", value: "G5" },
    { id: "g6", label: "Grade 6", value: "G6" },
    { id: "g7", label: "Grade 7", value: "G7" },
    { id: "g8", label: "Grade 8", value: "G8" },
    { id: "g9", label: "Grade 9", value: "G9" },
    { id: "g10", label: "Grade 10", value: "G10" },
    { id: "g11", label: "Grade 11", value: "G11" },
    { id: "g12", label: "Grade 12", value: "G12" },
];

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
        cell: (row) => <span className="text-muted-foreground">{row.grade_level}</span>,
    },
    {
        id: "stream_name",
        header: "Stream",
        width: "120px",
        cell: (row) => <span className="text-muted-foreground">{row.stream_name}</span>,
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
                submenu: GRADE_LEVEL_SUBMENU,
            },
        ];

        // Add stream filter if data is available
        const streams = streamsData?.items ?? [];
        if (streams.length > 0) {
            items.push({
                id: "stream_id",
                label: "Stream",
                icon: BookOpen,
                type: "sub_menu_multi",
                submenu: streams.map((s) => ({
                    id: s.id,
                    label: s.name,
                    value: s.id,
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
