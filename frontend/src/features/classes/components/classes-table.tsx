"use client";

import { useMemo } from "react";
import Link from "next/link";
import { DataTable } from "@/components/shared/data-table";
import { listClasses } from "../services/api";
import type { ClassListItem } from "../types/class";
import { useGrades } from "@/features/grades/hooks/use-grades";
import { useStreams } from "@/features/streams/hooks/use-streams";

function listClassesWithFilters(params: {
    page?: number;
    limit?: number;
    search?: string;
    filters?: Record<string, string | string[]>;
}) {
    return listClasses(params);
}

export function ClassesTable() {
    const { data: grades = [] } = useGrades();
    const { data: streams = [] } = useStreams();

    const gradeOptions = useMemo(
        () => grades.map((g) => ({ label: g.local_label, value: g.local_label, id: g.id })),
        [grades]
    );

    const streamOptions = useMemo(
        () => streams.map((s) => ({ label: s.name, value: s.name, id: s.id })),
        [streams]
    );

    const filterGroups = useMemo(
        () => [
            {
                id: "grade",
                label: "Grade Level",
                items: [
                    {
                        id: "grade-filter",
                        label: "Grade",
                        type: "sub_menu_multi" as const,
                        submenu:
                            gradeOptions.length > 0
                                ? gradeOptions.map((opt) => ({
                                      id: opt.id,
                                      label: opt.label,
                                      value: opt.value,
                                  }))
                                : [{ id: "empty", label: "No grades", value: "" }],
                    },
                ],
            },
            {
                id: "stream",
                label: "Stream",
                items: [
                    {
                        id: "stream-filter",
                        label: "Stream",
                        type: "sub_menu_multi" as const,
                        submenu:
                            streamOptions.length > 0
                                ? streamOptions.map((opt) => ({
                                      id: opt.id,
                                      label: opt.label,
                                      value: opt.value,
                                  }))
                                : [{ id: "empty", label: "No streams", value: "" }],
                    },
                ],
            },
        ],
        [gradeOptions, streamOptions]
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
