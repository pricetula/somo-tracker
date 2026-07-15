/**
 * InvoicesList — list of invoices with filter by status and term.
 */

"use client";

import Link from "next/link";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table";
import { getErrorMessage } from "@/lib/errors";
import { useInvoices } from "../hooks/use-finance-invoices";
import { useState } from "react";
import type { PaymentStatus } from "../types";

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

// ─── Component ────────────────────────────────────────────────────────────

export function InvoicesList() {
    const [statusFilter, setStatusFilter] = useState<string>("all");

    const filter: { payment_status?: PaymentStatus } = {};
    if (statusFilter !== "all") filter.payment_status = statusFilter as PaymentStatus;

    const { data, isLoading, isError, error } = useInvoices(filter);

    // ── Loading ──────────────────────────────────────────────────────────
    if (isLoading) {
        return (
            <div className="space-y-3">
                <Skeleton className="h-8 w-48" />
                <Skeleton className="h-10 w-full" />
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

    const invoices = data?.items ?? [];

    return (
        <div className="space-y-4">
            {/* ── Filters ────────────────────────────────────────────────── */}
            <div className="flex items-center gap-3">
                <Select value={statusFilter} onValueChange={setStatusFilter}>
                    <SelectTrigger className="w-40">
                        <SelectValue placeholder="Filter by status" />
                    </SelectTrigger>
                    <SelectContent>
                        <SelectItem value="all">All Statuses</SelectItem>
                        <SelectItem value="UNPAID">Unpaid</SelectItem>
                        <SelectItem value="PARTIAL">Partial</SelectItem>
                        <SelectItem value="PAID">Paid</SelectItem>
                        <SelectItem value="WAIVED">Waived</SelectItem>
                    </SelectContent>
                </Select>
                <p className="text-muted-foreground text-sm">
                    {invoices.length} invoice{invoices.length !== 1 ? "s" : ""}
                </p>
            </div>

            {/* ── Table ──────────────────────────────────────────────────── */}
            {invoices.length === 0 ? (
                <p className="text-muted-foreground text-sm">
                    No invoices found. Generate invoices to get started.
                </p>
            ) : (
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead>Student</TableHead>
                            <TableHead>Term</TableHead>
                            <TableHead>Status</TableHead>
                            <TableHead>Amount Due</TableHead>
                            <TableHead>Amount Paid</TableHead>
                            <TableHead className="text-right">Actions</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {invoices.map((inv) => (
                            <TableRow key={inv.id}>
                                <TableCell className="font-medium">
                                    {inv.student_id.slice(0, 8)}…
                                </TableCell>
                                <TableCell>{inv.academic_term_id.slice(0, 8)}…</TableCell>
                                <TableCell>{statusBadge(inv.payment_status)}</TableCell>
                                <TableCell>{inv.amount_due}</TableCell>
                                <TableCell>{inv.amount_paid}</TableCell>
                                <TableCell className="text-right">
                                    <Button variant="outline" size="sm" asChild>
                                        <Link href={`/finance/invoices/${inv.id}`}>View</Link>
                                    </Button>
                                </TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            )}
        </div>
    );
}
