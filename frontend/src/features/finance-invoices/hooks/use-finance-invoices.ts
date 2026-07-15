/**
 * TanStack Query hooks for invoices and payments.
 */

"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import {
    listInvoices,
    getInvoiceDetail,
    generateInvoice,
    waiveInvoice,
    recordPayment,
} from "@/lib/api/billing";
import { getErrorMessage } from "@/lib/errors";
import type {
    InvoiceFilter,
    GenerateInvoicePayload,
    RecordPaymentPayload,
} from "@/lib/api/billing";

// ─── Query keys ───────────────────────────────────────────────────────────

export const invoiceKeys = {
    all: ["invoices"] as const,
    lists: (filter?: InvoiceFilter) => [...invoiceKeys.all, "list", filter] as const,
    detail: (id: string) => [...invoiceKeys.all, "detail", id] as const,
};

// ─── Hooks — Queries ──────────────────────────────────────────────────────

export function useInvoices(filter: InvoiceFilter = {}) {
    return useQuery({
        queryKey: invoiceKeys.lists(filter),
        queryFn: () => listInvoices(filter),
        staleTime: 30 * 1000,
        placeholderData: (prev) => prev,
    });
}

export function useInvoiceDetail(id: string) {
    return useQuery({
        queryKey: invoiceKeys.detail(id),
        queryFn: () => getInvoiceDetail(id),
        staleTime: 30 * 1000,
        enabled: !!id,
    });
}

// ─── Hooks — Mutations ────────────────────────────────────────────────────

export function useGenerateInvoice() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: GenerateInvoicePayload) => generateInvoice(payload),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: invoiceKeys.all });
            toast.success("Invoice generated");
        },
        onError: (err) => toast.error(getErrorMessage(err)),
    });
}

export function useWaiveInvoice() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => waiveInvoice(id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: invoiceKeys.all });
            toast.success("Invoice waived");
        },
        onError: (err) => toast.error(getErrorMessage(err)),
    });
}

export function useRecordPayment() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: RecordPaymentPayload) => recordPayment(payload),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: invoiceKeys.all });
            toast.success("Payment recorded");
        },
        onError: (err) => toast.error(getErrorMessage(err)),
    });
}
