"use client";

import { useMemo } from "react";
import Link from "next/link";
import { DataTable } from "@/components/shared/data-table";
import { listClasses } from "../services/api";
import type { ClassListItem } from "../types/class";

function listClassesWithFilters(params: {
    page?: number;
    limit?: number;
    search?: string;
    filters?: Record<string, string | string[]>;
}) {
    return listClasses(params);
}

export function ClassesTable() {
    const filterGroups = useMemo(
        () => [
            {
                id: "grade",
                label: "Grade Level",
                items: [
                    {
                        id: "grade",
                        label: "Grade",
                        type: "sub_menu_multi" as const,
                        submenu: [
                            { id: "pp1", label: "PP1", value: "PP1" },
                            { id: "grade1", label: "Grade 1", value: "Grade 1" },
                            { id: "grade9", label: "Grade 9", value: "Grade 9" },
                            { id: "grade10", label: "Grade 10", value: "Grade 10" },
                            { id: "grade11", label: "Grade 11", value: "Grade 11" },
                            { id: "grade12", label: "Grade 12", value: "Grade 12" },
                        ],
                    },
                ],
            },
            {
                id: "stream",
                label: "Stream",
                items: [
                    {
                        id: "stream",
                        label: "Stream",
                        type: "sub_menu_multi" as const,
                        submenu: [
                            { id: "stem", label: "STEM", value: "STEM" },
                            { id: "arts", label: "Arts", value: "Arts" },
                            { id: "sports", label: "Sports Science", value: "Sports Science" },
                            { id: "social", label: "Social Science", value: "Social Science" },
                            { id: "general", label: "General", value: "General" },
                        ],
                    },
                ],
            },
        ],
        []
    );

    const columns = useMemo(
        () => [
            {
                id: "name",
                header: "Name",
                cell: (row: ClassListItem) => (
                    <Link
                        href={`/classes/${row.id}`}
                        className="underline underline-offset-4 hover:no-underline"
                    >
                        {row.name}
                    </Link>
                ),
                width: "2fr",
            },
            {
                id: "grade",
                header: "Grade",
                cell: (row: ClassListItem) => row.grade,
                width: "1fr",
            },
            {
                id: "stream",
                header: "Stream",
                cell: (row: ClassListItem) => row.stream,
                width: "1.5fr",
            },
        ],
        []
    );

    return (
        <DataTable<
            ClassListItem,
            { filters?: Record<string, string | string[]> },
            { items: ClassListItem[]; total: number }
        >
            queryKey={["classes"]}
            queryFn={listClassesWithFilters}
            getRowId={(row) => row.id}
            columns={columns}
            isSearchable
            searchPlaceholder="Search by name…"
            filterGroups={filterGroups}
            addHref="/classes/add"
            pageSize={50}
            height={600}
        />
    );
}
