"use client";

import { MemberInviteDialog } from "./member-invite-dialog";

export function ParentsInviteDialog() {
    return (
        <MemberInviteDialog
            role="PARENT"
            title="Invite Parents"
            description="Upload CSV or manually enter email addresses to invite new parents."
            closeHref="/parents"
        />
    );
}
