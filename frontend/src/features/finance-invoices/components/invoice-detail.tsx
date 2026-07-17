/**
 * InvoiceDetail — full invoice view with items, payments, and actions.
 *
 * Uses the shared DataTable component for line items and payment history.
 */

"use client";

import { useState } from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogTrigger,
} from "@/components/ui/dialog";
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
    AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import { getErrorMessage } from "@/lib/errors";
import { useInvoiceDetail, useWaiveInvoice, useRecordPayment } from "../hooks/use-finance-invoices";
import type { InvoiceItem, Payment } from "@/lib/api/billing";

// ─── Props ────────────────────────────────────────────────────────────────

interface InvoiceDetailProps {
    id: string;
}

// ─── Status helpers ───────────────────────────────────────────────────────

const statusColors: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
    UNPAID: "destructive",
    PARTIAL: "secondary",
    PAID: "default",
    WAIVED: "outline",
};

// ─── Line item columns ────────────────────────────────────────────────────

const lineItemColumns: DataTableColumn<InvoiceItem>[] = [
    {
        id: "description",
        header: "Description",
        cell: (row) => <span>{row.description || row.fee_category_id.slice(0, 8) + "…"}</span>,
    },
    {
        id: "amount",
        header: "Amount",
        width: "120px",
        align: "right",
        cell: (row) => <span className="font-medium tabular-nums">{row.amount}</span>,
    },
];

// ─── Payment columns ──────────────────────────────────────────────────────

const paymentColumns: DataTableColumn<Payment>[] = [
    {
        id: "amount",
        header: "Amount",
        cell: (row) => <span className="font-medium">{row.amount}</span>,
    },
    {
        id: "method",
        header: "Method",
        width: "120px",
        cell: (row) => <span className="text-muted-foreground">{row.payment_method || "—"}</span>,
    },
    {
        id: "reference",
        header: "Reference",
        width: "minmax(120px, 1fr)",
        cell: (row) => <span className="text-muted-foreground">{row.reference_code || "—"}</span>,
    },
    {
        id: "date",
        header: "Date",
        width: "120px",
        cell: (row) => (
            <span className="text-muted-foreground">
                {row.created_at ? new Date(row.created_at).toLocaleDateString() : "—"}
            </span>
        ),
    },
];

// ─── Component ────────────────────────────────────────────────────────────

export function InvoiceDetail({ id }: InvoiceDetailProps) {
    const { data, isLoading, isError, error } = useInvoiceDetail(id);
    const waiveMutation = useWaiveInvoice();
    const payMutation = useRecordPayment();

    // Payment dialog state
    const [payOpen, setPayOpen] = useState(false);
    const [payAmount, setPayAmount] = useState("");
    const [payMethod, setPayMethod] = useState("");

    // ── Loading ──────────────────────────────────────────────────────────
    if (isLoading) {
        return (
            <div className="space-y-4">
                <Skeleton className="h-8 w-48" />
                <Skeleton className="h-6 w-full" />
                <Skeleton className="h-20 w-full" />
            </div>
        );
    }

    // ── Error ────────────────────────────────────────────────────────────
    if (isError) {
        return (
            <Alert variant="destructive">
                <AlertDescription>{getErrorMessage(error)}</AlertDescription>
            </Alert>
        );
    }

    if (!data) {
        return (
            <Alert>
                <AlertDescription>Invoice not found.</AlertDescription>
            </Alert>
        );
    }

    const { invoice, items, payments } = data;
    const isPaidOrWaived = invoice.payment_status === "PAID" || invoice.payment_status === "WAIVED";
    const dueAmount = parseFloat(invoice.amount_due);
    const paidAmount = parseFloat(invoice.amount_paid);
    const remaining = Math.max(0, dueAmount - paidAmount);

    async function handleRecordPayment() {
        if (!payAmount) return;
        payMutation.mutate(
            {
                invoice_id: id,
                amount: payAmount,
                payment_method: payMethod || undefined,
            },
            {
                onSuccess: () => {
                    setPayOpen(false);
                    setPayAmount("");
                    setPayMethod("");
                },
            }
        );
    }

    return (
        <div className="space-y-6">
            {/* ── Header ──────────────────────────────────────────────── */}
            <div className="space-y-2">
                <div className="flex items-center gap-3">
                    <h1 className="text-foreground text-2xl font-semibold">Invoice</h1>
                    <Badge variant={statusColors[invoice.payment_status] ?? "outline"}>
                        {invoice.payment_status}
                    </Badge>
                    {invoice.invoice_label && (
                        <span className="text-muted-foreground">{invoice.invoice_label}</span>
                    )}
                </div>
                <p className="text-muted-foreground">
                    Student: {invoice.student_id.slice(0, 8)}… &mdash; Term:{" "}
                    {invoice.academic_term_id.slice(0, 8)}…
                </p>
            </div>

            {/* ── Summary Cards ──────────────────────────────────────────── */}
            <div className="flex gap-4">
                <div className="bg-muted/30 rounded-md px-4 py-3">
                    <p className="text-muted-foreground text-xs">Amount Due</p>
                    <p className="text-foreground text-xl font-semibold">{invoice.amount_due}</p>
                </div>
                <div className="bg-muted/30 rounded-md px-4 py-3">
                    <p className="text-muted-foreground text-xs">Amount Paid</p>
                    <p className="text-xl font-semibold text-emerald-600">{invoice.amount_paid}</p>
                </div>
                {remaining > 0 && invoice.payment_status !== "WAIVED" && (
                    <div className="bg-muted/30 rounded-md px-4 py-3">
                        <p className="text-muted-foreground text-xs">Remaining</p>
                        <p className="text-destructive text-xl font-semibold">
                            {remaining.toFixed(2)}
                        </p>
                    </div>
                )}
            </div>

            {/* ── Actions ──────────────────────────────────────────────── */}
            <div className="flex gap-2">
                {!isPaidOrWaived && remaining > 0 && (
                    <Dialog open={payOpen} onOpenChange={setPayOpen}>
                        <DialogTrigger asChild>
                            <Button size="sm">Record Payment</Button>
                        </DialogTrigger>
                        <DialogContent>
                            <DialogHeader>
                                <DialogTitle>Record Payment</DialogTitle>
                            </DialogHeader>
                            <div className="space-y-4 pt-2">
                                <div className="space-y-1.5">
                                    <Label>Amount</Label>
                                    <Input
                                        type="number"
                                        step="0.01"
                                        min="0"
                                        value={payAmount}
                                        onChange={(e) => setPayAmount(e.target.value)}
                                        placeholder="0.00"
                                    />
                                </div>
                                <div className="space-y-1.5">
                                    <Label>Payment Method (optional)</Label>
                                    <Input
                                        value={payMethod}
                                        onChange={(e) => setPayMethod(e.target.value)}
                                        placeholder="e.g. Cash, M-Pesa, Bank Transfer"
                                    />
                                </div>
                                {payMutation.error && (
                                    <p className="text-destructive">
                                        {getErrorMessage(payMutation.error)}
                                    </p>
                                )}
                                <div className="flex justify-end">
                                    <Button
                                        onClick={handleRecordPayment}
                                        disabled={!payAmount || payMutation.isPending}
                                    >
                                        {payMutation.isPending ? "Recording…" : "Record Payment"}
                                    </Button>
                                </div>
                            </div>
                        </DialogContent>
                    </Dialog>
                )}

                {invoice.payment_status !== "WAIVED" && (
                    <AlertDialog>
                        <AlertDialogTrigger asChild>
                            <Button variant="outline" size="sm">
                                Waive Invoice
                            </Button>
                        </AlertDialogTrigger>
                        <AlertDialogContent>
                            <AlertDialogHeader>
                                <AlertDialogTitle>Waive Invoice</AlertDialogTitle>
                                <AlertDialogDescription>
                                    Are you sure you want to waive this invoice? This will mark the
                                    full amount as waived and cannot be undone.
                                </AlertDialogDescription>
                            </AlertDialogHeader>
                            <AlertDialogFooter>
                                <AlertDialogCancel>Cancel</AlertDialogCancel>
                                <AlertDialogAction onClick={() => waiveMutation.mutate(id)}>
                                    Waive
                                </AlertDialogAction>
                            </AlertDialogFooter>
                        </AlertDialogContent>
                    </AlertDialog>
                )}
            </div>

            {/* ── Line Items ──────────────────────────────────────────────── */}
            <div className="space-y-3">
                <h2 className="text-foreground text-lg font-medium">Line Items</h2>
                {items.length === 0 ? (
                    <p className="text-muted-foreground">No line items.</p>
                ) : (
                    <DataTable
                        queryKey={["invoice-items", id]}
                        queryFn={() => Promise.resolve({ items, total: items.length })}
                        columns={lineItemColumns}
                        getRowId={(row) => row.id}
                        height={Math.min(items.length * 44 + 50, 300)}
                        pageSize={100}
                        emptyState="No line items."
                    />
                )}
            </div>

            {/* ── Payments ────────────────────────────────────────────────── */}
            <div className="space-y-3">
                <h2 className="text-foreground text-lg font-medium">
                    Payment History ({payments.length})
                </h2>
                {payments.length === 0 ? (
                    <p className="text-muted-foreground">No payments recorded.</p>
                ) : (
                    <DataTable
                        queryKey={["invoice-payments", id]}
                        queryFn={() => Promise.resolve({ items: payments, total: payments.length })}
                        columns={paymentColumns}
                        getRowId={(row) => row.id}
                        height={Math.min(payments.length * 44 + 50, 300)}
                        pageSize={100}
                        emptyState="No payments recorded."
                    />
                )}
            </div>
        </div>
    );
}
