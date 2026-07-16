/**
 * InvoiceDetail — full invoice view with items, payments, and actions.
 *
 * Actions: Record Payment, Waive Invoice.
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
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table";
import { getErrorMessage } from "@/lib/errors";
import { useInvoiceDetail, useWaiveInvoice, useRecordPayment } from "../hooks/use-finance-invoices";

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
                    <Table>
                        <TableHeader>
                            <TableRow>
                                <TableHead>Description</TableHead>
                                <TableHead className="text-right">Amount</TableHead>
                            </TableRow>
                        </TableHeader>
                        <TableBody>
                            {items.map((item) => (
                                <TableRow key={item.id}>
                                    <TableCell>
                                        {item.description || item.fee_category_id.slice(0, 8) + "…"}
                                    </TableCell>
                                    <TableCell className="text-right">{item.amount}</TableCell>
                                </TableRow>
                            ))}
                        </TableBody>
                    </Table>
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
                    <Table>
                        <TableHeader>
                            <TableRow>
                                <TableHead>Amount</TableHead>
                                <TableHead>Method</TableHead>
                                <TableHead>Reference</TableHead>
                                <TableHead>Date</TableHead>
                            </TableRow>
                        </TableHeader>
                        <TableBody>
                            {payments.map((p) => (
                                <TableRow key={p.id}>
                                    <TableCell className="font-medium">{p.amount}</TableCell>
                                    <TableCell>{p.payment_method || "—"}</TableCell>
                                    <TableCell>{p.reference_code || "—"}</TableCell>
                                    <TableCell className="text-muted-foreground">
                                        {p.created_at
                                            ? new Date(p.created_at).toLocaleDateString()
                                            : "—"}
                                    </TableCell>
                                </TableRow>
                            ))}
                        </TableBody>
                    </Table>
                )}
            </div>
        </div>
    );
}
