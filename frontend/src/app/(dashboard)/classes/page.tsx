/**
 * Classes listing page.
 *
 * Uses the shared DataTable component with filters for grade level and stream.
 * Stream options are fetched dynamically from the streams endpoint.
 * The current academic year and term are resolved server-side automatically.
 *
 * Maps to GET /api/v1/classes.
 */

"use client";

import Link from "next/link";
import { useMemo } from "react";
import { GraduationCap, Split } from "lucide-react";

import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import type { FilterGroup } from "@/components/shared/data-table/types";
import { listClasses, type Class } from "@/lib/api/classes";
import { StreamPill } from "@/features/settings-school";
import { GradeLevelPill, getGradeLevelFilterSubmenu } from "@/features/grade-level";
import { useStreamList } from "@/features/streams";
import { useDeleteClasses } from "@/features/classes";

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
    const { data: streamsData } = useStreamList();

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
                    label: "<StreamPill name={s.name} color={s.color} />",
                    value: s.id,
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
    }, [streamsData]);

    const deleteMutation = useDeleteClasses();

    return (
        <DataTable
            isCheckable
            addHref="/classes/add"
            queryKey={["classes"]}
            queryFn={listClasses}
            columns={columns}
            getRowId={(row) => row.id}
            isSearchable
            searchPlaceholder="Search by class name..."
            filterGroups={filterGroups}
            deleteFn={(id) => deleteMutation.mutateAsync([String(id)])}
            emptyState="No classes yet."
            noResultsState="No classes match your search or filters."
        />
    );
}
