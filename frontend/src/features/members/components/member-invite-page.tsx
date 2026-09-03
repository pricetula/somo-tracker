"use client";

import { BulkInviteForm } from "@/components/shared/bulk-invite";
import { submitBulkInvite, submitBulkParentInvite } from "@/lib/api/invitations";

/** Roles supported by BulkInviteForm. */
export type InviteRole = "SCHOOL_ADMIN" | "TEACHER" | "NURSE" | "FINANCE" | "PARENT";

interface MemberInvitePageProps {
    role: InviteRole;
    title?: string;
}

/**
 * Full-page variant of the bulk invite flow.
 *
 * Rendered by `/admins/add`, `/teachers/add`, ... page routes — i.e. when the
 * user hard-navigates or refreshes directly onto the add URL. The same form is
 * shown in a dialog (see `MemberInviteDialog`) when arriving via client-side
 * navigation.
 */
export function MemberInvitePage({ role }: MemberInvitePageProps) {
    const submitFn = role === "PARENT" ? submitBulkParentInvite : submitBulkInvite;
    return (
        <div className="space-y-6">
            <BulkInviteForm role={role} submitFn={submitFn} />
        </div>
    );
}
