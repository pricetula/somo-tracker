/**
 * Nurses listing page — active nurse staff.
 *
 * Uses the shared DataTable component with bulk delete and per-row actions.
 * Maps to GET /api/v1/members?role=NURSE.
 */

"use client";

import Link from "next/link";
import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import { RowActions } from "@/components/shared/data-table/row-actions";
import { Badge } from "@/components/ui/badge";
import { listNurses, type Member } from "@/lib/api/nurses";
import { InvitationCountBadge } from "@/features/invitations";
import { useDeleteNurse } from "@/features/nurses";

// ─── Columns factory ──────────────────────────────────────────────────────

function createColumns(
    deleteMutation: ReturnType<typeof useDeleteNurse>
): DataTableColumn<Member>[] {
    return [
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
        {
            id: "actions",
            header: "",
            width: "48px",
            align: "right",
            cell: (row) => (
                <RowActions
                    rowId={row.id}
                    label={row.full_name ?? row.email}
                    onDelete={() => deleteMutation.mutate(row.id)}
                    disabled={deleteMutation.isPending}
                />
            ),
        },
    ];
}

// ─── Page ──────────────────────────────────────────────────────────────────

export default function NursesPage() {
    const deleteMutation = useDeleteNurse();
    const columns = createColumns(deleteMutation);

    return (
        <DataTable
            isCheckable
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
            renderToolBarComponents={() => (
                <InvitationCountBadge
                    key="invitation-count"
                    role="NURSE"
                    href="/nurses/invitations"
                />
            )}
        />
    );
}
