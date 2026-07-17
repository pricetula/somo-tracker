/**
 * InvoicesList — list of invoices with filter by status.
 *
 * Uses the shared DataTable component with filter groups.
 */

"use client";

import Link from "next/link";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import type { FilterGroup } from "@/components/shared/data-table/types";
import { listInvoices } from "@/lib/api/billing";
import type { Invoice, PaymentStatus } from "@/lib/api/billing";

// ─── Status badge ─────────────────────────────────────────────────────────

const statusColors: Record<PaymentStatus, "default" | "secondary" | "destructive" | "outline"> = {
    UNPAID: "destructive",
    PARTIAL: "secondary",
    PAID: "default",
    WAIVED: "outline",
};

function statusBadge(status: PaymentStatus) {
    return <Badge variant={statusColors[status] ?? "outline"}>{status}</Badge>;
}

// ─── Columns ──────────────────────────────────────────────────────────────

const columns: DataTableColumn<Invoice>[] = [
    {
        id: "student_id",
        header: "Student",
        cell: (row) => <span className="font-medium">{row.student_id.slice(0, 8)}…</span>,
    },
    {
        id: "academic_term_id",
        header: "Term",
        width: "120px",
        cell: (row) => (
            <span className="text-muted-foreground">{row.academic_term_id.slice(0, 8)}…</span>
        ),
    },
    {
        id: "payment_status",
        header: "Status",
        width: "100px",
        cell: (row) => statusBadge(row.payment_status),
    },
    {
        id: "amount_due",
        header: "Amount Due",
        width: "120px",
        align: "right",
        cell: (row) => <span className="font-medium tabular-nums">{row.amount_due}</span>,
    },
    {
        id: "amount_paid",
        header: "Amount Paid",
        width: "120px",
        align: "right",
        cell: (row) => <span className="tabular-nums">{row.amount_paid}</span>,
    },
    {
        id: "actions",
        header: "",
        width: "80px",
        align: "right",
        cell: (row) => (
            <Button variant="outline" size="sm" asChild>
                <Link href={`/finance/invoices/${row.id}`}>View</Link>
            </Button>
        ),
    },
];

// ─── Filter Groups ────────────────────────────────────────────────────────

const filterGroups: FilterGroup[] = [
    {
        id: "invoice_filters",
        label: "Filter by",
        items: [
            {
                id: "payment_status",
                label: "Payment Status",
                type: "sub_menu_single",
                submenu: [
                    { id: "all", label: "All Statuses", value: "" },
                    { id: "UNPAID", label: "Unpaid", value: "UNPAID" },
                    { id: "PARTIAL", label: "Partial", value: "PARTIAL" },
                    { id: "PAID", label: "Paid", value: "PAID" },
                    { id: "WAIVED", label: "Waived", value: "WAIVED" },
                ],
            },
        ],
    },
];

// ─── Component ────────────────────────────────────────────────────────────

export function InvoicesList() {
    return (
        <DataTable
            queryKey={["invoices"]}
            queryFn={(
                params: { filters?: Record<string, string | string[]> } & {
                    page?: number;
                    limit?: number;
                }
            ) => {
                const filters = params.filters;
                const paymentStatus = filters?.payment_status as PaymentStatus | undefined;
                return listInvoices({
                    ...(paymentStatus ? { payment_status: paymentStatus } : {}),
                });
            }}
            columns={columns}
            getRowId={(row) => row.id}
            filterGroups={filterGroups}
            emptyState="No invoices found. Generate invoices to get started."
            noResultsState="No invoices match your filters."
            pageSize={50}
        />
    );
}
