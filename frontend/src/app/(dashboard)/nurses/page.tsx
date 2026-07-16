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
import Link from "next/link";
import { useDeleteNurse } from "@/features/nurses";

// ─── Columns ───────────────────────────────────────────────────────────────

const columns: DataTableColumn<Member>[] = [
    {
        id: "full_name",
        header: "Full Name",
        cell: (row) => (
            <Link href={`/nurses/${row.id}`} className="font-medium hover:underline">
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
            emptyState="No nurses yet."
            noResultsState="No nurses match your search."
        />
    );
}
