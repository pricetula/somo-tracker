/**
 * Nurses invitations listing page — shows all sent invitations for the NURSE role.
 *
 * Uses the shared DataTable component.
 * Maps to GET /api/v1/invitations?role=NURSE.
 *
 * Active staff are listed on the dedicated /nurses page.
 */

"use client";

import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import { Badge } from "@/components/ui/badge";
import { listInvitationsByRole, type Invitation } from "@/lib/api/invitations";

// ─── Status badge colours ─────────────────────────────────────────────────

const statusStyles: Record<string, string> = {
    pending: "bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400",
    accepted: "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400",
    expired: "bg-muted text-muted-foreground",
    revoked: "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400",
    invite_failed: "bg-destructive/10 text-destructive",
};

// ─── Columns ───────────────────────────────────────────────────────────────

const columns: DataTableColumn<Invitation>[] = [
    {
        id: "email",
        header: "Email",
        cell: (row) => <span className="font-medium">{row.email}</span>,
    },
    {
        id: "full_name",
        header: "Full Name",
        cell: (row) => row.full_name ?? "—",
    },
    {
        id: "status",
        header: "Status",
        width: "120px",
        cell: (row) => (
            <Badge
                variant="secondary"
                className={statusStyles[row.status] ?? "bg-muted text-muted-foreground"}
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
];

// ─── Page ──────────────────────────────────────────────────────────────────

export default function NursesInvitationsPage() {
    return (
        <DataTable
            queryKey={["invitations", "NURSE"]}
            queryFn={(params) => listInvitationsByRole({ ...params, role: "NURSE" })}
            columns={columns}
            getRowId={(row) => row.id}
            isSearchable
            searchPlaceholder="Search by email or name…"
            emptyState="No nurse invitations yet."
            noResultsState="No invitations match your search."
        />
    );
}
