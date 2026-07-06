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
import { useNurses } from "@/features/staff";
import { NursesTable } from "@/features/staff";

export default function NursesPage() {
    const [search, setSearch] = React.useState("");

    const { data, isLoading } = useNurses({ search: search || undefined });

    const nurses = data?.items ?? [];
    const total = data?.total ?? 0;

    return (
        <NursesTable
            nurses={nurses}
            total={total}
            isLoading={isLoading}
            search={search}
            onSearchChange={setSearch}
        />
    );
}
