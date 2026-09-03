"use client";

import { MemberInviteDialog } from "./member-invite-dialog";

export function FinanceInviteDialog() {
    return (
        <MemberInviteDialog
            role="FINANCE"
            title="Invite Finance Staff"
            description="Upload CSV or manually enter email addresses to invite new finance staff."
        />
    );
}
