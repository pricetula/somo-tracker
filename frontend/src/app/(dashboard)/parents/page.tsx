/**
 * Parents listing page.
 *
 * Shows all parent/guardian profiles with search and curriculum filter.
 * Uses the shared DataTable component.
 * Maps to GET /api/v1/parents.
 *
 * Curriculum filter filters parents whose linked children are in
 * selected education levels or grades.
 *
 * Bulk import is linked via DataTable's addHref → /parents/import.
 * Sent invitations are listed at /parents/invitations.
 */

"use client";

import Link from "next/link";
import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import { Badge } from "@/components/ui/badge";
import { listParents, type Parent } from "@/lib/api/parents";
import { CURRICULUM_FILTER_GROUPS } from "@/lib/curriculum-filters";
import { useDeleteParent } from "@/features/parents";

// ─── Columns ──────────────────────────────────────────────────────────────

const columns: DataTableColumn<Parent>[] = [
    {
        id: "full_name",
        header: "Full Name",
        cell: (row) => (
            <Link href={`/parents/${row.id}`} className="font-medium hover:underline">
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
        id: "phone_number",
        header: "Phone",
        cell: (row) => (
            <span className="text-muted-foreground font-mono">{row.phone_number || "—"}</span>
        ),
    },
    {
        id: "is_active",
        header: "Status",
        width: "100px",
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

// ─── Page ─────────────────────────────────────────────────────────────────

export default function ParentsPage() {
    const deleteMutation = useDeleteParent();

    return (
        <DataTable
            addHref="/parents/import"
            queryKey={["parents"]}
            queryFn={listParents}
            columns={columns}
            getRowId={(row) => row.id}
            isSearchable
            searchPlaceholder="Search by name or email…"
            filterGroups={CURRICULUM_FILTER_GROUPS}
            deleteFn={(id) => deleteMutation.mutateAsync(String(id))}
            emptyState="No parents yet."
            noResultsState="No parents match your search or filters."
        />
    );
}
