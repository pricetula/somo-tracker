/**
 * Nurses listing page — active nurse staff.
 *
 * Uses its own query hook and table component — not the generic
 * members module. Maps to GET /api/v1/members?role=NURSE.
 *
 * Invitations are listed on the dedicated /nurses/invitations page.
 */

"use client";

import * as React from "react";

import { NursesTable } from "@/features/staff/components/nurses-table";
import { useNurses } from "@/features/staff/hooks/use-nurses";

export default function NursesPage() {
    const {
        data: nursesData,
        isLoading: nursesLoading,
        isError: nursesError,
    } = useNurses({ includeInactive: true });

    return (
        <div className="flex flex-1 flex-col">
            {/* Page header */}
            <div className="flex items-center gap-3 px-6 pt-6 pb-2">
                <h1 className="text-2xl font-semibold tracking-tight">Nurses</h1>
            </div>

            <div className="flex flex-1 flex-col px-6 py-4">
                <section className="flex flex-1 flex-col">
                    {nursesError ? (
                        <div className="flex items-center justify-center py-8">
                            <p className="text-destructive text-sm">
                                Failed to load nurses. Please try again.
                            </p>
                        </div>
                    ) : (
                        <div className="ring-foreground/10 rounded-lg ring-1">
                            <NursesTable
                                nurses={nursesData?.members ?? []}
                                total={nursesData?.total ?? 0}
                                isLoading={nursesLoading}
                            />
                        </div>
                    )}
                </section>
            </div>
        </div>
    );
}
