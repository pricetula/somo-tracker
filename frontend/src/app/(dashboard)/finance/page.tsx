/**
 * Finance listing page — two independent, paginated lists stacked vertically:
 *   1. Active Staff (from GET /api/v1/members?role=FINANCE)
 *   2. Invited Staff (from GET /api/v1/invitations?role=FINANCE)
 */

"use client";

import { ActiveStaffTable, useStaffUsers } from "@/features/staff";

export default function FinancePage() {
    const {
        data: usersData,
        isLoading: usersLoading,
        isError: usersError,
    } = useStaffUsers("FINANCE");

    const roleLabel = "Finance";
    const addHref = "/finance/invitations";

    return (
        <section className="flex flex-col">
            {usersError ? (
                <div className="flex items-center justify-center py-8">
                    <p className="text-destructive text-sm">
                        Failed to load active finance staff. Please try again.
                    </p>
                </div>
            ) : (
                <div className="ring-foreground/10 rounded-lg ring-1">
                    <ActiveStaffTable
                        users={usersData?.members ?? []}
                        total={usersData?.total ?? 0}
                        roleLabel={roleLabel}
                        addHref={addHref}
                        isLoading={usersLoading}
                    />
                </div>
            )}
        </section>
    );
}
