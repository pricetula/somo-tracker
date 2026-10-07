"use client";

import Link from "next/link";
import { StaticTable } from "@/components/shared/static-table/static-table";
import { useTimetableTemplates } from "@/features/timetable/hooks/use-timetable-templates";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import type { TimetableTemplate } from "@/features/timetable/types/timetable-template";

export default function TimetablePage() {
    const { data: templates = [], isLoading } = useTimetableTemplates();

    const columns: DataTableColumn<TimetableTemplate>[] = [
        {
            id: "name",
            header: "Name",
            width: "1fr",
            cell: (row) => (
                <Link href={`/timetable/${row.id}`} className="font-medium">
                    {row.name}
                </Link>
            ),
        },
        {
            id: "description",
            header: "Description",
            width: "2fr",
            cell: (row) => <span className="text-muted-foreground">{row.description || "—"}</span>,
        },
        {
            id: "created_at",
            header: "Created",
            width: "140px",
            cell: (row) => (
                <span className="text-muted-foreground text-xs">
                    {row.created_at ? new Date(row.created_at).toLocaleDateString() : "—"}
                </span>
            ),
        },
    ];

    return (
        <div className="space-y-6 p-6">
            <h1 className="text-2xl font-semibold">Timetable Templates</h1>
            <StaticTable
                columns={columns}
                data={templates}
                getRowId={(row) => row.id}
                addHref="/timetable/add"
                isLoading={isLoading}
                emptyState={<span className="text-xs">No templates found.</span>}
            />
        </div>
    );
}
