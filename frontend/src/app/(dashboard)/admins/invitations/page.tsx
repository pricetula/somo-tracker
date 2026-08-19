/**
 * Admins invitations listing page.
 *
 * Shows all sent invitations for the SCHOOL_ADMIN role with revoke support.
 */

import { InvitationsList } from "@/features/invitations";

export default function AdminsInvitationsPage() {
    return (
        <InvitationsList
            role="SCHOOL_ADMIN"
            queryKey={["invitations", "SCHOOL_ADMIN"]}
            emptyState="No admin invitations yet."
        />
    );
}
