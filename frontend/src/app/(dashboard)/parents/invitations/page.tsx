/**
 * Parents invitations listing page — shows all sent parent invitations.
 *
 * Maps to GET /api/v1/invitations?role=PARENT.
 * Uses the shared DataTable component like the staff invitation pages.
 */

"use client";

import Link from "next/link";
import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Upload } from "lucide-react";
import { listInvitationsByRole, type Invitation } from "@/lib/api/invitations";

const statusVariantMap: Record<string, "secondary" | "outline" | "default" | "destructive"> = {
    pending: "secondary",
    accepted: "default",
    expired: "outline",
    revoked: "outline",
};

const statusLabelMap: Record<string, string> = {
    pending: "Pending",
    accepted: "Accepted",
    expired: "Expired",
    revoked: "Revoked",
};

// ─── Columns ──────────────────────────────────────────────────────────────

const columns: DataTableColumn<Invitation>[] = [
    {
        id: "email",
        header: "Email",
        cell: (row) => <span className="font-medium">{row.email}</span>,
    },
    {
        id: "full_name",
        header: "Full Name",
        cell: (row) => <span className="text-muted-foreground">{row.full_name || "—"}</span>,
    },
    {
        id: "status",
        header: "Status",
        width: "120px",
        cell: (row) => (
            <Badge variant={statusVariantMap[row.status] ?? "secondary"}>
                {statusLabelMap[row.status] ?? row.status}
            </Badge>
        ),
    },
    {
        id: "created_at",
        header: "Sent",
        width: "160px",
        cell: (row) => (
            <span className="text-muted-foreground text-xs">
                {new Date(row.created_at).toLocaleDateString(undefined, {
                    month: "short",
                    day: "numeric",
                    year: "numeric",
                })}
            </span>
        ),
    },
];

// ─── Page ─────────────────────────────────────────────────────────────────

export default function ParentsInvitationsPage() {
    return (
        <div className="flex flex-1 flex-col gap-4">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-semibold tracking-tight">Parent Invitations</h1>
                    <p className="text-muted-foreground mt-0.5 text-xs">
                        Sent invitations to parent/guardian email addresses.
                    </p>
                </div>
                <Button variant="outline" size="sm" asChild>
                    <Link href="/parents/import">
                        <Upload className="mr-1.5 size-3.5" />
                        Invite Parents
                    </Link>
                </Button>
            </div>
            <DataTable
                queryKey={["parents", "invitations"]}
                queryFn={({ page, limit }) =>
                    listInvitationsByRole({ role: "PARENT", page, limit })
                }
                columns={columns}
                getRowId={(row) => row.id}
                isSearchable
                searchPlaceholder="Search by email or name…"
                emptyState="No parent invitations sent yet."
                noResultsState="No invitations match your search."
            />
        </div>
    );
}
