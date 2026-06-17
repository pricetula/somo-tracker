/**
 * New Invitation Page (standalone route)
 *
 * Full-page form for creating multiple invitations.
 * When accessed via modal interception, the parallel route in @modal takes over.
 */

"use client";

import { InviteFormDialog } from "@/features/invitations";

export default function NewInvitationPage() {
    return <InviteFormDialog />;
}
