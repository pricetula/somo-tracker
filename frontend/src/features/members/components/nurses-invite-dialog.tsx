"use client";

import { MemberInviteDialog } from "./member-invite-dialog";

export function NursesInviteDialog() {
    return (
        <MemberInviteDialog
            role="NURSE"
            title="Invite Nurses"
            description="Upload CSV or manually enter email addresses to invite new nurses."
            closeHref="/nurses"
        />
    );
}
