/**
 * Nurses invitations listing page — shows all sent invitations for the NURSE role.
 *
 * Maps to GET /api/v1/invitations?role=NURSE.
 *
 * Active staff are listed on the dedicated /nurses page.
 */

"use client";

import * as React from "react";
import Link from "next/link";

import { InvitedStaffTable, useStaffInvitations } from "@/features/staff";
import { Button } from "@/components/ui/button";
import { UserPlus } from "lucide-react";

export default function NursesInvitationsPage() {
    const {
        data: invitationsData,
        isLoading: invitationsLoading,
        isError: invitationsError,
    } = useStaffInvitations("NURSE");

    const roleLabel = "Nurses";

    return (
        <div className="flex flex-1 flex-col">
            {/* Page header */}
            <div className="flex items-center gap-3 px-6 pt-6 pb-2">
                <h1 className="text-2xl font-semibold tracking-tight">Nurse Invitations</h1>
                <div className="ml-auto">
                    <Button size="sm" asChild>
                        <Link href="/nurses/invitations/new">
                            <UserPlus className="mr-1.5 size-3.5" />
                            Bulk Invite
                        </Link>
                    </Button>
                </div>
            </div>

            <div className="flex flex-1 flex-col px-6 py-4">
                <section className="flex flex-col">
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
