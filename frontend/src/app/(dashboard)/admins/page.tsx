/**
 * Admins listing page — active school administrators.
 *
 * Uses its own query hook and table component — not the generic
 * members module. Maps to GET /api/v1/members?role=SCHOOL_ADMIN.
 *
 * Invitations are listed on the dedicated /admins/invitations page.
 */

"use client";

import * as React from "react";
import { useAdmins } from "@/features/staff";
import { AdminsTable } from "@/features/staff";

export default function AdminsPage() {
    const [search, setSearch] = React.useState("");

    const { data, isLoading } = useAdmins({ search: search || undefined });

    const admins = data?.items ?? [];
    const total = data?.total ?? 0;

    return (
        <AdminsTable
            admins={admins}
            total={total}
            isLoading={isLoading}
            search={search}
            onSearchChange={setSearch}
        />
    );
}
