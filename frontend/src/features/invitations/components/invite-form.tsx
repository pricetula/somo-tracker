/**
 * Invite Form — reusable form for creating multiple invitations.
 *
 * Used both as:
 * - A dialog (in parallel intercepted route)
 * - A standalone page (direct navigation)
 *
 * Delegates to InviteFormDialogContent and InviteFormPageContent.
 */

"use client";

import { useRouter } from "next/navigation";

import { InviteFormDialogContent } from "./invite-form-dialog-content";
import { InviteFormPageContent } from "./invite-form-page-content";

// ─── Types ─────────────────────────────────────────────────────────────────

export interface InviteFormProps {
    /** When true, renders inside a Dialog (for modal interception). */
    asDialog?: boolean;
    open?: boolean;
    onOpenChange?: (open: boolean) => void;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function InviteFormDialog({ open, onOpenChange, asDialog }: InviteFormProps) {
    const router = useRouter();

    function handleSuccess() {
        if (onOpenChange) {
            onOpenChange(false);
        } else {
            router.back();
        }
    }

    // When used as a dialog (modal interception)
    if (asDialog) {
        return (
            <InviteFormDialogContent
                open={open ?? true}
                onOpenChange={onOpenChange ?? (() => router.back())}
                onSuccess={handleSuccess}
            />
        );
    }

    // Standalone page variant
    return <InviteFormPageContent onSuccess={handleSuccess} />;
}

export { InviteFormContent } from "./invite-form-content";
export { InviteFormDialogContent } from "./invite-form-dialog-content";
export { InviteFormPageContent } from "./invite-form-page-content";
