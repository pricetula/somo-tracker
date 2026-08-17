"use client";

import { MemberInviteDialog } from "./member-invite-dialog";

export function AdminsInviteDialog() {
    return (
        <MemberInviteDialog
            role="SCHOOL_ADMIN"
            title="Invite Admins"
            description="Upload CSV or manually enter email addresses to invite new admins."
        />
    );
}
