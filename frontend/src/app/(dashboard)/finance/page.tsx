/**
 * Finance listing page — active finance staff.
 *
 * Uses its own query hook and table component — not the generic
 * members module. Maps to GET /api/v1/members?role=FINANCE.
 *
 * Invitations are listed on the dedicated /finance/invitations page.
 */

"use client";

import * as React from "react";

import { FinanceTable } from "@/features/staff/components/finance-table";
import { useFinanceStaff } from "@/features/staff/hooks/use-finance";

export default function FinancePage() {
    const {
        data: financeData,
        isLoading: financeLoading,
        isError: financeError,
    } = useFinanceStaff({ includeInactive: true });

    return (
        <div className="flex flex-1 flex-col">
            {/* Page header */}
            <div className="flex items-center gap-3 px-6 pt-6 pb-2">
                <h1 className="text-2xl font-semibold tracking-tight">Finance Staff</h1>
            </div>

            <div className="flex flex-1 flex-col px-6 py-4">
                <section className="flex flex-1 flex-col">
                    {financeError ? (
                        <div className="flex items-center justify-center py-8">
                            <p className="text-destructive text-sm">
                                Failed to load finance staff. Please try again.
                            </p>
                        </div>
                    ) : (
                        <div className="ring-foreground/10 rounded-lg ring-1">
                            <FinanceTable
                                staff={financeData?.members ?? []}
                                total={financeData?.total ?? 0}
                                isLoading={financeLoading}
                            />
                        </div>
                    )}
                </section>
            </div>
        </div>
    );
}
