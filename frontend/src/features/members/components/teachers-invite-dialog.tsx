"use client";

import { MemberInviteDialog } from "./member-invite-dialog";

export function TeachersInviteDialog() {
    return (
        <MemberInviteDialog
            role="TEACHER"
            title="Invite Teachers"
            description="Upload CSV or manually enter email addresses to invite new teachers."
        />
    );
}
