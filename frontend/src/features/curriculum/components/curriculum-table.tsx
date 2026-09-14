"use client";

import { useMemo } from "react";
import Link from "next/link";
import { DataTable } from "@/components/shared/data-table";
import { listSubjects } from "../services/api";
import type { SubjectListItem } from "../types/curriculum";

function listSubjectsWithFilters(params: {
    page?: number;
    limit?: number;
    search?: string;
    filters?: Record<string, string | string[]>;
}) {
    return listSubjects(params);
}

export function CurriculumTable() {
    const filterGroups = useMemo(
        () => [
            {
                id: "grade",
                label: "Grade",
                items: [
                    {
                        id: "grade",
                        label: "Grade",
                        type: "sub_menu_single" as const,
                        submenu: [
                            { id: "all", label: "All", value: "all" },
                            { id: "pp1", label: "PP1", value: "PP1" },
                            { id: "grade1", label: "Grade 1", value: "Grade 1" },
                            { id: "grade10", label: "Grade 10", value: "Grade 10" },
                            { id: "grade12", label: "Grade 12", value: "Grade 12" },
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
                cell: (row: SubjectListItem) => (
                    <Link
                        href={`/curriculum/${row.id}`}
                        className="underline underline-offset-4 hover:no-underline"
                    >
                        {row.name}
                    </Link>
                ),
                width: "2fr",
            },
            {
                id: "code",
                header: "Code",
                cell: (row: SubjectListItem) => row.code,
                width: "1.5fr",
            },
            {
                id: "grade",
                header: "Grade",
                cell: (row: SubjectListItem) => row.grade,
                width: "1fr",
            },
        ],
        []
    );

    return (
        <DataTable<
            SubjectListItem,
            { filters?: Record<string, string | string[]> },
            { items: SubjectListItem[]; total: number }
        >
            queryKey={["subjects"]}
            queryFn={listSubjectsWithFilters}
            getRowId={(row) => row.id}
            columns={columns}
            isSearchable
            searchPlaceholder="Search by name or code…"
            filterGroups={filterGroups}
            addHref="/curriculum/add"
            pageSize={50}
            height={600}
        />
    );
}
