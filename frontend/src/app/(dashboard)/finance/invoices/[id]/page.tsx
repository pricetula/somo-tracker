/**
 * Invoice detail page — view items, payments, record payments, waive.
 */

import { InvoiceDetail } from "@/features/finance-invoices";

interface InvoiceDetailPageProps {
    params: Promise<{ id: string }>;
}

export default async function InvoiceDetailPage({ params }: InvoiceDetailPageProps) {
    const { id } = await params;
    return <InvoiceDetail id={id} />;
}
