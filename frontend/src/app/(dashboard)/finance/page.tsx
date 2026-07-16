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
import Link from "next/link";
import { useDeleteFinanceStaff } from "@/features/finance";

// ─── Columns ───────────────────────────────────────────────────────────────

const columns: DataTableColumn<Member>[] = [
    {
        id: "full_name",
        header: "Full Name",
        cell: (row) => (
            <Link href={`/finance/${row.id}`} className="font-medium hover:underline">
                {row.full_name || "—"}
            </Link>
        ),
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

// ─── Page ──────────────────────────────────────────────────────────────────

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
            emptyState="No finance staff yet."
            noResultsState="No finance staff match your search."
        />
    );
}
