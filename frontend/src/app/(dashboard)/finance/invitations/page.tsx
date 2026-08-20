/**
 * Finance invitations listing page.
 *
 * Shows all sent invitations for the FINANCE role with revoke support.
 */

import { InvitationsList } from "@/features/invitations";

export default function FinanceInvitationsPage() {
    return (
        <InvitationsList
            role="FINANCE"
            queryKey={["invitations", "FINANCE"]}
            emptyState="No finance invitations yet."
        />
    );
}
