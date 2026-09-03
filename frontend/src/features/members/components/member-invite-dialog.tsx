"use client";

import React from "react";
import { useRouter } from "next/navigation";
import { BulkInviteForm } from "@/components/shared/bulk-invite";
import { submitBulkInvite, submitBulkParentInvite } from "@/lib/api/invitations";
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
}

/**
 * Dialog variant of the bulk invite flow.
 *
 * Rendered by the `@modal` parallel slot's intercepted route (e.g.
 * `(dashboard)/@modal/(.)admins/add/page.tsx`), which only matches during
 * client-side navigation. On a hard navigation/refresh the intercept is not
 * matched and the full-page variant renders instead.
 */
export function MemberInviteDialog({ role, title, description }: MemberInviteDialogProps) {
    const router = useRouter();

    const handleRouteBack = React.useCallback(() => {
        router.back();
    }, [router]);

    const submitFn = role === "PARENT" ? submitBulkParentInvite : submitBulkInvite;

    return (
        <Dialog open onOpenChange={handleRouteBack}>
            <DialogContent className="w-full sm:max-w-xl">
                <DialogHeader>
                    <DialogTitle>{title}</DialogTitle>
                    <DialogDescription>{description}</DialogDescription>
                </DialogHeader>
                <BulkInviteForm role={role} submitFn={submitFn} onSuccess={handleRouteBack} />
            </DialogContent>
        </Dialog>
    );
}
