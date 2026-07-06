/**
 * Nurses listing page — active nurse staff.
 *
 * Uses the shared DataTable component.
 * Maps to GET /api/v1/members?role=NURSE.
 *
 * Invitations are listed on the dedicated /nurses/invitations page.
 */

"use client";

import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import { Badge } from "@/components/ui/badge";
import { listNurses, type Member } from "@/lib/api/nurses";
import { useDeleteNurse } from "@/features/staff";

const columns: DataTableColumn<Member>[] = [
    {
        id: "full_name",
        header: "Full Name",
        cell: (row) => <span className="text-sm font-medium">{row.full_name || "—"}</span>,
    },
    {
        id: "email",
        header: "Email",
        cell: (row) => <span className="text-muted-foreground text-sm">{row.email}</span>,
    },
    {
        id: "is_active",
        header: "Account Status",
        cell: (row) => (
            <Badge
                variant="secondary"
                className={
                    row.is_active
                        ? "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400"
                        : "bg-muted text-muted-foreground"
                }
            >
                {row.is_active ? "Active" : "Inactive"}
            </Badge>
        ),
    },
];

export default function NursesPage() {
    const deleteMutation = useDeleteNurse();

    return (
        <DataTable
            addHref="/nurses/import"
            queryKey={["nurses"]}
            queryFn={listNurses}
            columns={columns}
            getRowId={(row) => row.id}
            isSearchable
            searchPlaceholder="Search by name or email…"
            deleteFn={(id) => deleteMutation.mutateAsync(String(id))}
            rowHeight={48}
            height={600}
            emptyState="No nurses yet."
            noResultsState="No nurses match your search."
        />
    );
}
