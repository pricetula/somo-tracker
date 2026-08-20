/**
 * Finance Staff Detail Page — Full page render for /finance/:id.
 *
 * On hard refresh, this renders the finance staff detail view directly.
 * When client-navigated from the finance table, it renders inside
 * the dashboard layout along with the modal slot.
 */

import { FinanceDetail } from "@/features/finance";

interface Props {
    params: Promise<{ id: string }>;
}

export default async function FinanceDetailPage({ params }: Props) {
    const { id } = await params;
    return (
        <div className="p-6">
            <FinanceDetail id={id} />
        </div>
    );
}
