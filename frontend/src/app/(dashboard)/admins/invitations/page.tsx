/**
 * Invitations Page
 *
 * Lists all invitations with filters: search by name, email, status, role, expired.
 */

"use client";

import * as React from "react";
import Link from "next/link";

import { InvitationTable, useInvitations } from "@/features/invitations";
import { Button } from "@/components/ui/button";
import { Mail } from "lucide-react";
import type { InvitationStatus, InvitationRole } from "@/lib/api/invitations";

export default function InvitationsPage() {
    const [search, setSearch] = React.useState("");
    const [debouncedSearch, setDebouncedSearch] = React.useState("");
    const [emailFilter, setEmailFilter] = React.useState("");
    const [debouncedEmail, setDebouncedEmail] = React.useState("");
    const [statusFilter, setStatusFilter] = React.useState<InvitationStatus | "">("");
    const [roleFilter, setRoleFilter] = React.useState<InvitationRole | "">("");
    const [showExpired, setShowExpired] = React.useState(false);

    // Debounce search inputs
    React.useEffect(() => {
        const timer = setTimeout(() => setDebouncedSearch(search), 300);
        return () => clearTimeout(timer);
    }, [search]);

    React.useEffect(() => {
        const timer = setTimeout(() => setDebouncedEmail(emailFilter), 300);
        return () => clearTimeout(timer);
    }, [emailFilter]);

    const { data, isLoading } = useInvitations({
        search: debouncedSearch,
        email: debouncedEmail,
        status: statusFilter || undefined,
        role: roleFilter || undefined,
        expired: showExpired ? undefined : false,
        page: 1,
        per_page: 50,
    });

    const invitations = data?.invitations ?? [];
    const total = data?.total ?? 0;

    return (
        <div className="flex flex-1 flex-col">
            <div className="flex items-center gap-3 px-6 pt-6 pb-2">
                <h1 className="text-2xl font-semibold tracking-tight">Invitations</h1>
                {total > 0 && (
                    <span className="border-border/40 text-muted-foreground inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium">
                        {total}
                    </span>
                )}
                <div className="ml-auto">
                    <Button asChild size="sm">
                        <Link href="/admins/invitations/new">
                            <Mail className="mr-1.5 size-3.5" />
                            Invite Users
                        </Link>
                    </Button>
                </div>
            </div>

            <InvitationTable
                invitations={invitations}
                total={total}
                search={search}
                onSearchChange={setSearch}
                emailFilter={emailFilter}
                onEmailFilterChange={setEmailFilter}
                statusFilter={statusFilter}
                onStatusFilterChange={setStatusFilter}
                roleFilter={roleFilter}
                onRoleFilterChange={setRoleFilter}
                showExpired={showExpired}
                onShowExpiredChange={setShowExpired}
                onInviteClick={() => {
                    window.location.href = "/admins/invitations/new";
                }}
                isLoading={isLoading}
            />
        </div>
    );
}
