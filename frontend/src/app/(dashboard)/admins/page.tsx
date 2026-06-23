/**
 * Admins listing page — shows pending invitations for SCHOOL_ADMIN role.
 *
 * NOTE: The backend does not expose an endpoint to list active SCHOOL_ADMIN
 * members (GET /api/v1/members only supports NURSE, FINANCE, TEACHER).
 * Only the invitations table is shown here.
 */

"use client";

import { InvitedStaffTable, useStaffInvitations } from "@/features/staff";

export default function AdminsPage() {
    const {
        data: invitationsData,
        isLoading: invitationsLoading,
        isError: invitationsError,
    } = useStaffInvitations("SCHOOL_ADMIN");

    const roleLabel = "Admins";

    return (
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
    );
}
