/**
 * Admins listing page — active school administrators.
 *
 * Uses the shared DataTable component.
 * Maps to GET /api/v1/members?role=SCHOOL_ADMIN.
 *
 * Invitations are listed on the dedicated /admins/invitations page.
 */

"use client";

import { useQuery } from "@tanstack/react-query";
import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { listAdmins, type Member } from "@/lib/api/admins";
import { getInvitationCount } from "@/lib/api/invitations";
import { useDeleteAdmin } from "@/features/admin";
import Link from "next/link";

// ─── Columns ───────────────────────────────────────────────────────────────

const columns: DataTableColumn<Member>[] = [
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
];

// ─── Invitation Count Badge ────────────────────────────────────────────────

function InvitationCountBadge() {
    const { data, isLoading } = useQuery({
        queryKey: ["invitations", "count", "SCHOOL_ADMIN"],
        queryFn: () => getInvitationCount("SCHOOL_ADMIN"),
    });

    if (isLoading) {
        return <Skeleton className="h-9 w-28" />;
    }

    const count = data?.total ?? 0;
    const label = `${count} ${count === 1 ? "invitation" : "invitations"}`;

    return (
        <Button variant="outline" size="sm" asChild>
            <Link href="/admins/invitations">{label}</Link>
        </Button>
    );
}

// ─── Page ──────────────────────────────────────────────────────────────────

export default function AdminsPage() {
    const deleteMutation = useDeleteAdmin();

    return (
        <DataTable
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
            toolBarComponents={[<InvitationCountBadge key="invitation-count" />]}
        />
    );
}
