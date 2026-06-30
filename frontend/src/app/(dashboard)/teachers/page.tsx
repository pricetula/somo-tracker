/**
 * Teachers listing page — active staff members for the TEACHER role.
 *
 * Maps to GET /api/v1/members?role=TEACHER.
 *
 * Invitations are listed on the dedicated /teachers/invitations page.
 */

"use client";

import * as React from "react";

import { ActiveStaffTable, useStaffUsers } from "@/features/staff";

export default function TeachersPage() {
    const {
        data: usersData,
        isLoading: usersLoading,
        isError: usersError,
    } = useStaffUsers("TEACHER");

    const roleLabel = "Teachers";

    return (
        <div className="flex flex-1 flex-col">
            {/* Page header */}
            <div className="flex items-center gap-3 px-6 pt-6 pb-2">
                <h1 className="text-2xl font-semibold tracking-tight">Teachers</h1>
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
                                addHref="/teachers/invitations"
                                isLoading={usersLoading}
                            />
                        </div>
                    )}
                </section>
            </div>
        </div>
    );
}
