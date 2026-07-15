/**
 * Finance listing page — active finance staff.
 *
 * Uses the shared DataTable component.
 * Maps to GET /api/v1/members?role=FINANCE.
 *
 * Invitations are listed on the dedicated /finance/invitations page.
 */

"use client";

import { useState } from "react";
import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Pencil } from "lucide-react";
import { listFinanceStaff, type Member } from "@/lib/api/finance";
import { useDeleteFinanceStaff } from "@/features/staff";
import { MemberEditDialog } from "@/features/staff/components/member-edit-dialog";

// ─── Edit action cell ──────────────────────────────────────────────────────

function EditCell({ row }: { row: Member }) {
    const [editOpen, setEditOpen] = useState(false);
    return (
        <>
            <Button
                variant="ghost"
                size="icon-sm"
                onClick={() => setEditOpen(true)}
                title="Edit finance staff"
            >
                <Pencil className="h-4 w-4" />
                <span className="sr-only">Edit {row.full_name}</span>
            </Button>
            <MemberEditDialog
                userId={row.id}
                open={editOpen}
                onOpenChange={setEditOpen}
                invalidationKey={["finance"]}
            />
        </>
    );
}

// ─── Columns ───────────────────────────────────────────────────────────────

const columns: DataTableColumn<Member>[] = [
    {
        id: "full_name",
        header: "Full Name",
        cell: (row) => <span className="font-medium">{row.full_name || "—"}</span>,
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
];

const editActionColumn: DataTableColumn<Member> = {
    id: "edit",
    header: "",
    width: "48px",
    align: "right",
    cell: (row) => <EditCell row={row} />,
};

// ─── Page ──────────────────────────────────────────────────────────────────

export default function FinancePage() {
    const deleteMutation = useDeleteFinanceStaff();

    return (
        <DataTable
            addHref="/finance/import"
            queryKey={["finance"]}
            queryFn={listFinanceStaff}
            columns={[...columns, editActionColumn]}
            getRowId={(row) => row.id}
            isSearchable
            searchPlaceholder="Search by name or email…"
            deleteFn={(id) => deleteMutation.mutateAsync(String(id))}
            emptyState="No finance staff yet."
            noResultsState="No finance staff match your search."
        />
    );
}
