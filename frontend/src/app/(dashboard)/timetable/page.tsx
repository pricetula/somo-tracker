/**
 * Timetable Track List — /timetable
 * Shows all timetable tracks for the active school.
 */
"use client";

import React from "react";
import Link from "next/link";
import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import { getTracks, type TimetableTrack } from "@/lib/api/timetable";

const columns: DataTableColumn<TimetableTrack>[] = [
    {
        id: "name",
        header: "Name",
        width: "1fr",
        cell: (row) => (
            <Link
                href={`/timetable/${row.id}`}
                className="text-foreground hover:text-primary font-medium hover:underline"
            >
                {row.name}
            </Link>
        ),
    },
    {
        id: "description",
        header: "Description",
        width: "2fr",
        cell: (row) => (
            <span className="text-muted-foreground block truncate">
                {row.description || "No description"}
            </span>
        ),
    },
    {
        id: "is_default",
        header: "Default",
        width: "100px",
        cell: (row) => <span className="text-xs">{row.is_default ? "Yes" : "—"}</span>,
    },
];

export default function TimetableListPage() {
    return (
        <div className="space-y-6 p-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-semibold tracking-tight">Timetables</h1>
                    <p className="text-muted-foreground mt-1">Track schedules for your school.</p>
                </div>
            </div>

            <DataTable
                queryKey={["timetable", "tracks"]}
                queryFn={getTracks}
                columns={columns}
                getRowId={(row) => row.id}
                addHref="/timetable/new"
                isSearchable={true}
                searchPlaceholder="Search timetables..."
                emptyState={
                    <div className="rounded-xl border p-12 text-center">
                        <p className="text-muted-foreground">No timetables yet.</p>
                        <Link href="/timetable/new" className="mt-4 inline-block">
                            <button className="bg-primary text-primary-foreground hover:bg-primary/90 inline-flex items-center justify-center gap-2 rounded-md px-3 py-2 text-sm font-medium">
                                Create first timetable
                            </button>
                        </Link>
                    </div>
                }
            />
        </div>
    );
}
