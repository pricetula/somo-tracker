/**
 * Staff Directory Page
 *
 * Lists support staff with virtual table and bulk invite via Stytch.
 */

"use client";

import * as React from "react";

import { MemberTable, BulkInviteModal, EmptyState, useMembers } from "@/features/members";

const ROLE = "SUPPORT_STAFF" as const;
const ROLE_LABEL = "Staff";

export default function StaffPage() {
    const [search, setSearch] = React.useState("");
    const [debouncedSearch, setDebouncedSearch] = React.useState("");
    const [inviteOpen, setInviteOpen] = React.useState(false);

    React.useEffect(() => {
        const timer = setTimeout(() => {
            setDebouncedSearch(search);
        }, 300);
        return () => clearTimeout(timer);
    }, [search]);

    const { data, isLoading } = useMembers(ROLE, {
        search: debouncedSearch,
    });

    const members = data?.members ?? [];
    const total = data?.total ?? 0;
    const isEmpty = !isLoading && total === 0 && !debouncedSearch;

    return (
        <div className="flex flex-1 flex-col">
            <div className="flex items-center gap-3 px-6 pt-6 pb-2">
                <h1 className="text-2xl font-semibold tracking-tight">Staff</h1>
                {total > 0 && (
                    <span className="border-border/40 text-muted-foreground inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium">
                        {total}
                    </span>
                )}
            </div>

            {isEmpty ? (
                <EmptyState roleLabel={ROLE_LABEL} onInvite={() => setInviteOpen(true)} />
            ) : (
                <MemberTable
                    members={members}
                    total={total}
                    roleLabel={ROLE_LABEL}
                    search={search}
                    onSearchChange={setSearch}
                    onInviteClick={() => setInviteOpen(true)}
                    isLoading={isLoading}
                />
            )}

            <BulkInviteModal open={inviteOpen} onOpenChange={setInviteOpen} role={ROLE} />
        </div>
    );
}
