/**
 * Nurses invitations listing page.
 *
 * Shows all sent invitations for the NURSE role with revoke support.
 */

import { InvitationsList } from "@/features/invitations";

export default function NursesInvitationsPage() {
    return (
        <InvitationsList
            role="NURSE"
            queryKey={["invitations", "NURSE"]}
            emptyState="No nurse invitations yet."
        />
    );
}
