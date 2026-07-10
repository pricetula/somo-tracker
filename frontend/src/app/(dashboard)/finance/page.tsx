/**
 * Finance listing page — active finance staff.
 *
 * Uses the shared DataTable component.
 * Maps to GET /api/v1/members?role=FINANCE.
 *
 * Invitations are listed on the dedicated /finance/invitations page.
 */

"use client";

import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import { Badge } from "@/components/ui/badge";
import { listFinanceStaff, type Member } from "@/lib/api/finance";
import { useDeleteFinanceStaff } from "@/features/staff";

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

export default function FinancePage() {
    const deleteMutation = useDeleteFinanceStaff();

    return (
        <DataTable
            addHref="/finance/import"
            queryKey={["finance"]}
            queryFn={listFinanceStaff}
            columns={columns}
            getRowId={(row) => row.id}
            isSearchable
            searchPlaceholder="Search by name or email…"
            deleteFn={(id) => deleteMutation.mutateAsync(String(id))}
            rowHeight={48}
            height={600}
            emptyState="No finance staff yet."
            noResultsState="No finance staff match your search."
        />
    );
}
