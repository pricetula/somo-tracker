/**
 * Intercepted route — Finance invitations list rendered as a sliding side sheet.
 *
 * When a user clicks the invitation count badge in the finance table toolbar,
 * this sheet slides out from the right showing the FINANCE invitations list.
 * On hard refresh the full page at /finance/invitations takes over.
 */

"use client";

import { useRouter } from "next/navigation";
import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import { Badge } from "@/components/ui/badge";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
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
        width: "140px",
        cell: (row) => (
            <span className="text-muted-foreground">
                {new Date(row.created_at).toLocaleDateString()}
            </span>
        ),
    },
];

// ─── Sheet ─────────────────────────────────────────────────────────────────

export default function FinanceInvitationsSheet() {
    const router = useRouter();

    return (
        <Sheet
            open
            onOpenChange={(open) => {
                if (!open) router.back();
            }}
        >
            <SheetContent side="right" className="w-full data-[side=right]:sm:max-w-xl">
                <SheetHeader>
                    <SheetTitle>Finance Invitations</SheetTitle>
                </SheetHeader>
                <div className="flex-1 overflow-y-auto px-6 pb-6">
                    <DataTable
                        queryKey={["invitations", "FINANCE"]}
                        queryFn={(params) => listInvitationsByRole({ ...params, role: "FINANCE" })}
                        columns={columns}
                        getRowId={(row) => row.id}
                        isSearchable
                        searchPlaceholder="Search by email or name…"
                        emptyState="No finance invitations yet."
                        noResultsState="No invitations match your search."
                        height={480}
                    />
                </div>
            </SheetContent>
        </Sheet>
    );
}
