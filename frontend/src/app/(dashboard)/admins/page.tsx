/**
 * Admins listing page — active school administrators.
 *
 * Uses the shared DataTable component.
 * Maps to GET /api/v1/members?role=SCHOOL_ADMIN.
 *
 * Invitations are listed on the dedicated /admins/invitations page.
 */

"use client";

import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import { Badge } from "@/components/ui/badge";
import { listAdmins, type Member } from "@/lib/api/admins";
import { useDeleteAdmin } from "@/features/staff";

const columns: DataTableColumn<Member>[] = [
    {
        id: "full_name",
        header: "Full Name",
        cell: (row) => <span className="font-medium">{row.full_name || "—"}</span>,
    },
    {
        id: "email",
        header: "Email",
        cell: (row) => <span className="text-muted-foreground">{row.email}</span>,
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

export default function AdminsPage() {
    const deleteMutation = useDeleteAdmin();

    return (
        <DataTable
            addHref="/admins/import"
            queryKey={["admins"]}
            queryFn={listAdmins}
            columns={columns}
            getRowId={(row) => row.id}
            isSearchable
            searchPlaceholder="Search by name or email…"
            deleteFn={(id) => deleteMutation.mutateAsync(String(id))}
            rowHeight={48}
            height={600}
            emptyState="No admins yet."
            noResultsState="No admins match your search."
        />
    );
}
