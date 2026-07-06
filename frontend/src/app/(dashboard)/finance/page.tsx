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
import { useFinanceStaff } from "@/features/staff";
import { FinanceTable } from "@/features/staff";

export default function FinancePage() {
    const [search, setSearch] = React.useState("");

    const { data, isLoading } = useFinanceStaff({ search: search || undefined });

    const staff = data?.items ?? [];
    const total = data?.total ?? 0;

    return (
        <FinanceTable
            staff={staff}
            total={total}
            isLoading={isLoading}
            search={search}
            onSearchChange={setSearch}
        />
    );
}
