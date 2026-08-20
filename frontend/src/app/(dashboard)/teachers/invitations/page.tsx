/**
 * Teachers invitations listing page.
 *
 * Shows all sent invitations for the TEACHER role with revoke support.
 */

import { InvitationsList } from "@/features/invitations";

export default function TeachersInvitationsPage() {
    return (
        <InvitationsList
            role="TEACHER"
            queryKey={["invitations", "TEACHER"]}
            emptyState="No teacher invitations yet."
        />
    );
}
