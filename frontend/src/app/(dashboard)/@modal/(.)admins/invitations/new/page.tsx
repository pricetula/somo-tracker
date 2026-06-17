/**
 * Intercepted New Invitation Modal
 *
 * When navigating from /admins/invitations, the "new" route is intercepted
 * and rendered as a dialog overlay via the @modal parallel route.
 */

"use client";

import { useRouter } from "next/navigation";
import { InviteFormDialog } from "@/features/invitations";

export default function NewInvitationModal() {
    const router = useRouter();

    return (
        <InviteFormDialog
            asDialog
            open
            onOpenChange={(open) => {
                if (!open) router.back();
            }}
        />
    );
}
