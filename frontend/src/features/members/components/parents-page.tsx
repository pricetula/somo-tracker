"use client";

import Link from "next/link";
import { DataTable } from "@/components/shared/data-table/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import { Badge } from "@/components/ui/badge";
import {
    listParents,
    deleteParent,
    type Parent,
    type ListParentsParams,
    type ListParentsResponse,
} from "@/lib/api/parents";
import { RowActions } from "@/components/shared/data-table/row-actions";
import { InvitationCountBadge } from "@/features/invitations";
import { useDeleteParent } from "@/features/parents";

function createColumns(
    deleteMutation: ReturnType<typeof useDeleteParent>
): DataTableColumn<Parent>[] {
    return [
        {
            id: "full_name",
            header: "Full Name",
            cell: (row) => <Link href={`/parents/${row.id}`}>{row.full_name || "—"}</Link>,
        },
        {
            id: "email",
            header: "Email",
            cell: (row) => <span className="text-muted-foreground">{row.email}</span>,
        },
        {
            id: "phone",
            header: "Phone",
            cell: (row) => <span className="text-muted-foreground">{row.phone_number || "-"}</span>,
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

export function ParentsPage() {
    const deleteMutation = useDeleteParent();
    const columns = createColumns(deleteMutation);

    return (
        <DataTable<Parent, ListParentsParams, ListParentsResponse>
            addHref="/parents/add"
            queryKey={["parents"]}
            queryFn={listParents}
            params={{}}
            columns={columns}
            getRowId={(row) => row.id}
            isSearchable
            searchPlaceholder="Search parents..."
            isCheckable
            deleteFn={(id) => deleteParent(String(id))}
            renderToolBarComponents={() => (
                <InvitationCountBadge
                    key="invitation-count"
                    role="PARENT"
                    href="/parents/invitations"
                />
            )}
        />
    );
}
