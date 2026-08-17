"use client";

import { useRouter } from "next/navigation";
import { BulkInviteForm } from "@/components/shared/bulk-invite";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
} from "@/components/ui/dialog";
import type { InviteRole } from "./member-invite-page";

interface MemberInviteDialogProps {
    role: InviteRole;
    title: string;
    description: string;
    /** Where to navigate when the dialog is dismissed. */
    closeHref: string;
}

/**
 * Dialog variant of the bulk invite flow.
 *
 * Rendered by the `@modal` parallel slot's intercepted route (e.g.
 * `(dashboard)/@modal/(.)admins/add/page.tsx`), which only matches during
 * client-side navigation. On a hard navigation/refresh the intercept is not
 * matched and the full-page variant renders instead.
 */
export function MemberInviteDialog({
    role,
    title,
    description,
    closeHref,
}: MemberInviteDialogProps) {
    const router = useRouter();

    const handleClose = () => {
        router.push(closeHref);
    };

    return (
        <Dialog
            open
            onOpenChange={(next) => {
                if (!next) handleClose();
            }}
        >
            <DialogContent className="w-full max-w-md">
                <DialogHeader>
                    <DialogTitle>{title}</DialogTitle>
                    <DialogDescription>{description}</DialogDescription>
                </DialogHeader>
                <BulkInviteForm role={role} />
            </DialogContent>
        </Dialog>
    );
}
