"use client";

import { FinanceTable } from "@/features/finance";

export default function FinancePage() {
    return (
        <div className="space-y-4 p-6">
            <h1 className="text-2xl font-semibold">Finance</h1>
            <FinanceTable />
        </div>
    );
}
