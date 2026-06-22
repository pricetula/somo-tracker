/**
 * Admins listing page — shows pending invitations for SCHOOL_ADMIN role.
 *
 * NOTE: The backend does not expose an endpoint to list active SCHOOL_ADMIN
 * members (GET /api/v1/members only supports NURSE, FINANCE, TEACHER).
 * Only the invitations table is shown here.
 */

"use client";

import * as React from "react";
import Link from "next/link";

import { InvitedStaffTable, useStaffInvitations } from "@/features/staff";
import { Button } from "@/components/ui/button";
import { UserPlus } from "lucide-react";

export default function AdminsPage() {
    const {
        data: invitationsData,
        isLoading: invitationsLoading,
        isError: invitationsError,
    } = useStaffInvitations("SCHOOL_ADMIN");

    const roleLabel = "Admins";
    const addHref = "./add";

    return (
        <div className="flex flex-1 flex-col">
            {/* Page header */}
            <div className="flex items-center gap-3 px-6 pt-6 pb-2">
                <h1 className="text-2xl font-semibold tracking-tight">Admins</h1>
                <div className="ml-auto">
                    <Button size="sm" asChild>
                        <Link href={addHref}>
                            <UserPlus className="mr-1.5 size-3.5" />
                            Invite Admins
                        </Link>
                    </Button>
                </div>
            </div>

            <div className="flex flex-1 flex-col gap-8 px-6 py-4">
                {/* Invitations */}
                <section className="flex flex-col">
                    <h2 className="mb-2 text-sm font-medium">Invited {roleLabel}</h2>
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
