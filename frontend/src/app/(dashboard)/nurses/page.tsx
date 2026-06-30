/**
 * Nurses listing page — active staff members for the NURSE role.
 *
 * Maps to GET /api/v1/members?role=NURSE.
 *
 * Invitations are listed on the dedicated /nurses/invitations page.
 */

"use client";

import * as React from "react";

import { ActiveStaffTable, useStaffUsers } from "@/features/staff";

export default function NursesPage() {
    const {
        data: usersData,
        isLoading: usersLoading,
        isError: usersError,
    } = useStaffUsers("NURSE");

    const roleLabel = "Nurses";

    return (
        <div className="flex flex-1 flex-col">
            {/* Page header */}
            <div className="flex items-center gap-3 px-6 pt-6 pb-2">
                <h1 className="text-2xl font-semibold tracking-tight">Nurses</h1>
            </div>

            <div className="flex flex-1 flex-col px-6 py-4">
                <section className="flex flex-col">
                    {usersError ? (
                        <div className="flex items-center justify-center py-8">
                            <p className="text-destructive text-sm">
                                Failed to load active {roleLabel.toLowerCase()}. Please try again.
                            </p>
                        </div>
                    ) : (
                        <div className="ring-foreground/10 rounded-lg ring-1">
                            <ActiveStaffTable
                                users={usersData?.members ?? []}
                                total={usersData?.total ?? 0}
                                roleLabel={roleLabel}
                                addHref="/nurses/invitations"
                                isLoading={usersLoading}
                            />
                        </div>
                    )}
                </section>
            </div>
        </div>
    );
}
