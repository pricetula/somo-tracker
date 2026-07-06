/**
 * Parents listing page.
 *
 * Shows all parent/guardian profiles with search and curriculum filter.
 * Uses the shared DataTable component.
 * Maps to GET /api/v1/parents.
 *
 * Curriculum filter filters parents whose linked children are in
 * selected education levels or grades.
 */

"use client";

import Link from "next/link";
import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Upload } from "lucide-react";
import { listParents, type Parent } from "@/lib/api/parents";
import { CURRICULUM_FILTER_GROUPS } from "@/lib/curriculum-filters";
import { useDeleteParent } from "@/features/parents";

// ─── Columns ──────────────────────────────────────────────────────────────

const columns: DataTableColumn<Parent>[] = [
    {
        id: "full_name",
        header: "Full Name",
        cell: (row) => (
            <Link href={`/parents/${row.id}`} className="text-sm font-medium hover:underline">
                {row.full_name || "—"}
            </Link>
        ),
    },
    {
        id: "email",
        header: "Email",
        cell: (row) => <span className="text-muted-foreground text-sm">{row.email}</span>,
    },
    {
        id: "phone_number",
        header: "Phone",
        cell: (row) => (
            <span className="text-muted-foreground font-mono text-sm">
                {row.phone_number || "—"}
            </span>
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
        <div className="flex flex-1 flex-col gap-4">
            <div className="flex items-center justify-between">
                <h1 className="text-2xl font-semibold tracking-tight">Parents</h1>
                <Button variant="outline" size="sm" asChild>
                    <Link href="/parents/import">
                        <Upload className="mr-1.5 size-3.5" />
                        Bulk Import
                    </Link>
                </Button>
            </div>
            <DataTable
                queryKey={["parents"]}
                queryFn={listParents}
                columns={columns}
                getRowId={(row) => row.id}
                isSearchable
                searchPlaceholder="Search by name or email…"
                filterGroups={CURRICULUM_FILTER_GROUPS}
                deleteFn={(id) => deleteMutation.mutateAsync(String(id))}
                rowHeight={48}
                height={600}
                emptyState="No parents yet."
                noResultsState="No parents match your search or filters."
            />
        </div>
    );
}
