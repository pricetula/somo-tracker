/**
 * InvoicesList — list of invoices with filter by status.
 *
 * Uses the shared DataTable component with filter groups.
 */

"use client";

import Link from "next/link";
import { FileX } from "lucide-react";
import { toast } from "sonner";
import { useQueryClient } from "@tanstack/react-query";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import type { FilterGroup } from "@/components/shared/data-table/types";
import { RowActions } from "@/components/shared/data-table/row-actions";
import type { RowAction } from "@/components/shared/data-table/row-actions";
import { listInvoices, waiveInvoice } from "@/lib/api/billing";
import type { Invoice, PaymentStatus } from "@/lib/api/billing";
import { getErrorMessage } from "@/lib/errors";

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

// ─── Waivable statuses ────────────────────────────────────────────────────

const WAIVABLE_STATUSES: PaymentStatus[] = ["UNPAID", "PARTIAL"];

// ─── Columns factory ──────────────────────────────────────────────────────

function createColumns(queryClient: ReturnType<typeof useQueryClient>): DataTableColumn<Invoice>[] {
    return [
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
            width: "120px",
            align: "right",
            cell: (row) => {
                const waivable = WAIVABLE_STATUSES.includes(row.payment_status);
                const rowActions: RowAction[] = waivable
                    ? [
                          {
                              label: "Waive",
                              icon: FileX,
                              destructive: true,
                              confirmTitle: "Waive Invoice",
                              confirmDescription: `Are you sure you want to waive this invoice? The amount due of ${row.amount_due} will be forgiven.`,
                              onClick: async () => {
                                  try {
                                      await waiveInvoice(row.id);
                                      queryClient.invalidateQueries({ queryKey: ["invoices"] });
                                      toast.success("Invoice waived.");
                                  } catch (err) {
                                      toast.error(getErrorMessage(err));
                                  }
                              },
                          },
                      ]
                    : [];

                return (
                    <div className="flex items-center justify-end gap-1">
                        <Button variant="outline" size="sm" asChild>
                            <Link href={`/finance/invoices/${row.id}`}>View</Link>
                        </Button>
                        {waivable && (
                            <RowActions rowId={row.id} label="invoice" actions={rowActions} />
                        )}
                    </div>
                );
            },
        },
    ];
}

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
    const queryClient = useQueryClient();
    const columns = createColumns(queryClient);

    return (
        <DataTable
            isCheckable
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
