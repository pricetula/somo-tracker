/**
 * Admins listing page — active school administrators.
 *
 * Uses the shared DataTable component with bulk delete and per-row actions.
 * Maps to GET /api/v1/members?role=SCHOOL_ADMIN.
 */

"use client";

import Link from "next/link";
import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import { RowActions } from "@/components/shared/data-table/row-actions";
import { Badge } from "@/components/ui/badge";
import { listAdmins, type Member } from "@/lib/api/admins";
import { InvitationCountBadge } from "@/features/invitations";
import { useDeleteAdmin } from "@/features/admin";

// ─── Columns factory ──────────────────────────────────────────────────────

function createColumns(
    deleteMutation: ReturnType<typeof useDeleteAdmin>
): DataTableColumn<Member>[] {
    return [
        {
            id: "full_name",
            header: "Full Name",
            cell: (row) => <Link href={`/admins/${row.id}`}>{row.full_name || "—"}</Link>,
        },
        {
            id: "email",
            header: "Email",
            cell: (row) => <span className="text-muted-foreground">{row.email}</span>,
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

// ─── Page ─────────────────────────────────────────────────────────────────

export default function AdminsPage() {
    const deleteMutation = useDeleteAdmin();
    const columns = createColumns(deleteMutation);

    return (
        <DataTable
            isCheckable
            addHref="/admins/import"
            queryKey={["admins"]}
            queryFn={listAdmins}
            columns={columns}
            getRowId={(row) => row.id}
            isSearchable
            searchPlaceholder="Search by name or email…"
            deleteFn={(id) => deleteMutation.mutateAsync(String(id))}
            emptyState="No admins yet."
            noResultsState="No admins match your search."
            renderToolBarComponents={() => (
                <InvitationCountBadge
                    key="invitation-count"
                    role="SCHOOL_ADMIN"
                    href="/admins/invitations"
                />
            )}
        />
    );
}
