/**
 * Billing API functions — fee categories, fee templates, invoices, payments.
 *
 * Backend: backend/internal/billing/handler.go
 *
 * Endpoints:
 *   Fee Categories:  POST/GET/PUT/DELETE /api/v1/billing/fee-categories
 *   Fee Templates:   POST/GET/PUT/DELETE /api/v1/billing/fee-templates
 *   Invoices:        POST /api/v1/billing/invoices/generate
 *                    GET  /api/v1/billing/invoices
 *                    GET  /api/v1/billing/invoices/:id
 *                    POST /api/v1/billing/invoices/:id/waive
 *   Payments:        POST /api/v1/billing/payments
 *                    GET  /api/v1/billing/payments?invoice_id=...
 */

import { api } from "./client";

// ─── Fee Category ─────────────────────────────────────────────────────────

export interface FeeCategory {
    id: string;
    tenant_id: string;
    school_id: string;
    name: string;
    is_mandatory: boolean;
}

export interface CreateFeeCategoryPayload {
    name: string;
    is_mandatory: boolean;
}

export interface UpdateFeeCategoryPayload {
    name?: string;
    is_mandatory?: boolean;
}

export interface ListFeeCategoriesResponse {
    items: FeeCategory[];
    total: number;
}

// ─── Fee Template ─────────────────────────────────────────────────────────

export interface FeeTemplate {
    id: string;
    tenant_id: string;
    school_id: string;
    academic_term_id: string;
    grade_level: string;
    fee_category_id: string;
    amount: string;
    created_at: string;
}

export interface CreateFeeTemplatePayload {
    grade_level: string;
    fee_category_id: string;
    amount: string;
}

export interface UpdateFeeTemplatePayload {
    amount?: string;
}

export interface ListFeeTemplatesResponse {
    items: FeeTemplate[];
    total: number;
}

// ─── Invoice ──────────────────────────────────────────────────────────────

export type PaymentStatus = "UNPAID" | "PARTIAL" | "PAID" | "WAIVED";

export interface Invoice {
    id: string;
    student_id: string;
    academic_term_id: string;
    parent_id?: string | null;
    invoice_label?: string | null;
    payment_status: PaymentStatus;
    amount_due: string;
    amount_paid: string;
    created_at: string;
}

export interface InvoiceItem {
    id: string;
    invoice_id: string;
    fee_category_id: string;
    description?: string;
    amount: string;
}

export interface Payment {
    id: string;
    invoice_id: string;
    amount: string;
    parent_id?: string | null;
    payment_method?: string | null;
    reference_code?: string | null;
    recorded_by: string;
    created_at: string;
}

export interface InvoiceDetailResponse {
    invoice: Invoice;
    items: InvoiceItem[];
    payments: Payment[];
}

export interface InvoiceItemInput {
    fee_category_id: string;
    description?: string;
    amount: string;
}

export interface GenerateInvoicePayload {
    student_id: string;
    academic_term_id: string;
    invoice_label?: string;
    items?: InvoiceItemInput[];
}

export interface ListInvoicesResponse {
    items: Invoice[];
    total: number;
}

export interface InvoiceFilter {
    student_id?: string;
    academic_term_id?: string;
    payment_status?: PaymentStatus;
}

// ─── Payment ──────────────────────────────────────────────────────────────

export interface RecordPaymentPayload {
    invoice_id: string;
    amount: string;
    parent_id?: string;
    payment_method?: string;
    reference_code?: string;
}

export interface ListPaymentsResponse {
    items: Payment[];
    total: number;
}

// ==========================================================================
// API Functions — Fee Categories
// ==========================================================================

export async function listFeeCategories(): Promise<ListFeeCategoriesResponse> {
    return api.get<ListFeeCategoriesResponse>("/api/v1/billing/fee-categories");
}

export async function createFeeCategory(
    payload: CreateFeeCategoryPayload
): Promise<{ id: string }> {
    return api.post<{ id: string }>("/api/v1/billing/fee-categories", payload);
}

export async function updateFeeCategory(
    id: string,
    payload: UpdateFeeCategoryPayload
): Promise<void> {
    await api.put(`/api/v1/billing/fee-categories/${id}`, payload);
}

export async function deleteFeeCategory(id: string): Promise<void> {
    await api.delete(`/api/v1/billing/fee-categories`, { id });
}

// ==========================================================================
// API Functions — Fee Templates
// ==========================================================================

export async function listFeeTemplates(
    params: { grade_level?: string } = {}
): Promise<ListFeeTemplatesResponse> {
    const searchParams = new URLSearchParams();
    if (params.grade_level) searchParams.set("grade_level", params.grade_level);
    const qs = searchParams.toString();

    return api.get<ListFeeTemplatesResponse>(`/api/v1/billing/fee-templates?${qs}`);
}

export async function createFeeTemplate(
    payload: CreateFeeTemplatePayload
): Promise<{ id: string }> {
    return api.post<{ id: string }>("/api/v1/billing/fee-templates", payload);
}

export async function updateFeeTemplate(
    id: string,
    payload: UpdateFeeTemplatePayload
): Promise<void> {
    await api.put(`/api/v1/billing/fee-templates/${id}`, payload);
}

export async function deleteFeeTemplate(id: string): Promise<void> {
    await api.delete(`/api/v1/billing/fee-templates`, { id });
}

// ==========================================================================
// API Functions — Invoices
// ==========================================================================

export async function listInvoices(filter: InvoiceFilter = {}): Promise<ListInvoicesResponse> {
    const searchParams = new URLSearchParams();
    if (filter.student_id) searchParams.set("student_id", filter.student_id);
    if (filter.payment_status) searchParams.set("payment_status", filter.payment_status);
    const qs = searchParams.toString();

    return api.get<ListInvoicesResponse>(`/api/v1/billing/invoices?${qs}`);
}

export async function getInvoiceDetail(id: string): Promise<InvoiceDetailResponse> {
    return api.get<InvoiceDetailResponse>(`/api/v1/billing/invoices/${id}`);
}

export async function generateInvoice(
    payload: GenerateInvoicePayload
): Promise<InvoiceDetailResponse> {
    return api.post<InvoiceDetailResponse>("/api/v1/billing/invoices/generate", payload);
}

export async function waiveInvoice(id: string): Promise<void> {
    await api.post(`/api/v1/billing/invoices/${id}/waive`);
}

// ==========================================================================
// API Functions — Payments
// ==========================================================================

export async function recordPayment(payload: RecordPaymentPayload): Promise<{ id: string }> {
    return api.post<{ id: string }>("/api/v1/billing/payments", payload);
}

export async function listPayments(invoiceId: string): Promise<ListPaymentsResponse> {
    return api.get<ListPaymentsResponse>(`/api/v1/billing/payments?invoice_id=${invoiceId}`);
}
