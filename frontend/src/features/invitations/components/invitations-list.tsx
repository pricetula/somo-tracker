"use client";

import { DataTable } from "@/components/shared/data-table";
import { type DataTableColumn } from "@/components/shared/data-table/types";
import { Badge } from "@/components/ui/badge";
import { listInvitationsByRole, revokeInvitation, type Invitation } from "@/lib/api/invitations";

const STATUS_STYLES: Record<string, string> = {
    pending: "bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400",
    accepted: "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400",
    expired: "bg-muted text-muted-foreground",
    revoked: "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400",
    invite_failed: "bg-destructive/10 text-destructive",
};
function createColumns(queryKey: readonly unknown[]): DataTableColumn<Invitation>[] {
    return [
        {
            id: "email",
            header: "Email",
            cell: (row) => <span className="font-medium">{row.email}</span>,
        },
        {
            id: "full_name",
            header: "Full Name",
            cell: (row) => row.full_name ?? <span className="text-muted-foreground">—</span>,
        },
        {
            id: "status",
            header: "Status",
            width: "120px",
            cell: (row) => (
                <Badge
                    variant="secondary"
                    className={STATUS_STYLES[row.status] ?? "bg-muted text-muted-foreground"}
                >
                    {row.status.replace("_", " ")}
                </Badge>
            ),
        },
        {
            id: "created_at",
            header: "Sent",
            width: "160px",
            cell: (row) => (
                <span className="text-muted-foreground">
                    {new Date(row.created_at).toLocaleDateString()}
                </span>
            ),
        },
        {
            id: "actions",
            header: "",
            width: "48px",
            align: "right",
            cell: (row) => <RevokeCell invitation={row} queryKey={queryKey} />,
        },
    ];
}
interface InvitationsListProps {
    role: string;
    queryKey: readonly unknown[];
    emptyState: string;
}

import { RevokeCell } from "./revoke-cell";

export function InvitationsList({ role, queryKey, emptyState }: InvitationsListProps) {
    const columns = createColumns(queryKey);

    return (
        <DataTable
            isCheckable
            queryKey={queryKey}
            queryFn={(params) => listInvitationsByRole({ ...params, role })}
            columns={columns}
            getRowId={(row) => row.id}
            isSearchable
            searchPlaceholder="Search by email or name…"
            deleteFn={(id) => revokeInvitation(String(id))}
            emptyState={emptyState}
            noResultsState="No invitations match your search."
        />
    );
}
