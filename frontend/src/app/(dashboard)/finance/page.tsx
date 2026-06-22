/**
 * Finance listing page — two independent, paginated lists stacked vertically:
 *   1. Active Staff (from GET /api/v1/users?role=FINANCE)
 *   2. Invited Staff (from GET /api/v1/invitations?role=FINANCE&status[]=...)
 */

"use client";

import * as React from "react";
import Link from "next/link";

import {
    ActiveStaffTable,
    InvitedStaffTable,
    useStaffUsers,
    useStaffInvitations,
} from "@/features/staff";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { UserPlus } from "lucide-react";

export default function FinancePage() {
    const {
        data: usersData,
        isLoading: usersLoading,
        isError: usersError,
    } = useStaffUsers("FINANCE");

    const {
        data: invitationsData,
        isLoading: invitationsLoading,
        isError: invitationsError,
    } = useStaffInvitations("FINANCE");

    const roleLabel = "Finance";
    const addHref = "./add";

    return (
        <div className="flex flex-1 flex-col">
            {/* Page header */}
            <div className="flex items-center gap-3 px-6 pt-6 pb-2">
                <h1 className="text-2xl font-semibold tracking-tight">Finance</h1>
                <div className="ml-auto">
                    <Button size="sm" asChild>
                        <Link href={addHref}>
                            <UserPlus className="mr-1.5 size-3.5" />
                            Invite Finance Staff
                        </Link>
                    </Button>
                </div>
            </div>

            <div className="flex flex-1 flex-col gap-8 px-6 py-4">
                {/* List 1 — Active Staff */}
                <section className="flex flex-col">
                    <h2 className="mb-2 text-sm font-medium">Active Finance Staff</h2>
                    {usersError ? (
                        <div className="flex items-center justify-center py-8">
                            <p className="text-destructive text-sm">
                                Failed to load active finance staff. Please try again.
                            </p>
                        </div>
                    ) : (
                        <div className="ring-foreground/10 rounded-lg ring-1">
                            <ActiveStaffTable
                                users={usersData?.users ?? []}
                                total={usersData?.total ?? 0}
                                roleLabel={roleLabel}
                                addHref={addHref}
                                isLoading={usersLoading}
                            />
                        </div>
                    )}
                </section>

                <Separator />

                {/* List 2 — Invitations */}
                <section className="flex flex-col">
                    <h2 className="mb-2 text-sm font-medium">Invited Finance Staff</h2>
                    {invitationsError ? (
                        <div className="flex items-center justify-center py-8">
                            <p className="text-destructive text-sm">
                                Failed to load invitations. Please try again.
                            </p>
                        </div>
                    ) : (
                        <div className="ring-foreground/10 rounded-lg ring-1">
                            <InvitedStaffTable
                                invitations={invitationsData?.invitations ?? []}
                                total={invitationsData?.total ?? 0}
                                roleLabel={roleLabel}
                                isLoading={invitationsLoading}
                            />
                        </div>
                    )}
                </section>
            </div>
        </div>
    );
}
