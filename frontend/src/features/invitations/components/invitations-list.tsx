/**
 * InvitationsList — shared DataTable listing for sent invitations with revoke support.
 *
 * Used by all invitation pages (admins, teachers, nurses, finance, parents).
 * Provides bulk revoke (via checkboxes) and per-row revoke (via dropdown).
 */

"use client";

import { toast } from "sonner";
import { useQueryClient } from "@tanstack/react-query";
import { XCircle } from "lucide-react";

import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import { RowActions } from "@/components/shared/data-table/row-actions";
import type { RowAction } from "@/components/shared/data-table/row-actions";
import { Badge } from "@/components/ui/badge";
import { listInvitationsByRole, revokeInvitation, type Invitation } from "@/lib/api/invitations";
import { getErrorMessage } from "@/lib/errors";

// ─── Status badge styles ──────────────────────────────────────────────────

const STATUS_STYLES: Record<string, string> = {
    pending: "bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400",
    accepted: "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400",
    expired: "bg-muted text-muted-foreground",
    revoked: "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400",
    invite_failed: "bg-destructive/10 text-destructive",
};

// ─── Revoke Cell ──────────────────────────────────────────────────────────

function RevokeCell({
    invitation,
    queryKey,
}: {
    invitation: Invitation;
    queryKey: readonly unknown[];
}) {
    const queryClient = useQueryClient();

    if (invitation.status !== "pending") {
        return <div className="w-12" />;
    }

    const rowActions: RowAction[] = [
        {
            label: "Revoke",
            icon: XCircle,
            destructive: true,
            confirmTitle: "Revoke Invitation",
            confirmDescription: `Are you sure you want to revoke the invitation for "${invitation.email}"? They will no longer be able to accept it.`,
            onClick: async () => {
                try {
                    await revokeInvitation(invitation.id);
                    queryClient.invalidateQueries({ queryKey });
                    toast.success("Invitation revoked.");
                } catch (err) {
                    toast.error(getErrorMessage(err));
                }
            },
        },
    ];

    return <RowActions rowId={invitation.id} label={invitation.email} actions={rowActions} />;
}

// ─── Columns factory ──────────────────────────────────────────────────────

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

// ─── Props ────────────────────────────────────────────────────────────────

interface InvitationsListProps {
    role: string;
    queryKey: readonly unknown[];
    emptyState: string;
}

// ─── InvitationsList ──────────────────────────────────────────────────────

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
