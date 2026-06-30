/**
 * Teachers listing page — active teachers with extended educator fields.
 *
 * Uses its own dedicated teachers endpoint (GET /api/v1/teachers) with
 * TSC Number, KNEC Panel Assessor ID, and Core Assignment Role.
 *
 * Invitations are listed on the dedicated /teachers/invitations page.
 */

"use client";

import * as React from "react";

import { TeachersTable } from "@/features/staff/components/teachers-table";
import { useTeachers } from "@/features/staff/hooks/use-teachers";

export default function TeachersPage() {
    const {
        data: teachersData,
        isLoading: teachersLoading,
        isError: teachersError,
    } = useTeachers({ includeInactive: true });

    return (
        <div className="flex flex-1 flex-col">
            {/* Page header */}
            <div className="flex items-center gap-3 px-6 pt-6 pb-2">
                <h1 className="text-2xl font-semibold tracking-tight">Teachers</h1>
            </div>

            <div className="flex flex-1 flex-col px-6 py-4">
                <section className="flex flex-1 flex-col">
                    {teachersError ? (
                        <div className="flex items-center justify-center py-8">
                            <p className="text-destructive text-sm">
                                Failed to load teachers. Please try again.
                            </p>
                        </div>
                    ) : (
                        <div className="ring-foreground/10 rounded-lg ring-1">
                            <TeachersTable
                                teachers={teachersData?.teachers ?? []}
                                total={teachersData?.total ?? 0}
                                isLoading={teachersLoading}
                            />
                        </div>
                    )}
                </section>
            </div>
        </div>
    );
}
