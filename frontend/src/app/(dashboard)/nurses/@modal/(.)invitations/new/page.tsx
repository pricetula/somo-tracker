/**
 * Intercepted /nurses/invitations/new — renders as a dialog overlay via the @modal slot.
 *
 * When users navigate to /nurses/invitations/new from within /nurses (Link click, soft nav),
 * Next.js intercepts the route and renders this page inside the @modal parallel slot,
 * overlaying the /nurses/invitations listing page without unmounting it.
 *
 * Mirrors the pattern established by /nurses/@modal/(.)add/page.tsx.
 */

"use client";

import { useRouter } from "next/navigation";

import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { BulkStaffImport } from "@/features/staff-import";

export default function InterceptedNursesBulkInvite() {
    const router = useRouter();

    function handleClose() {
        router.back();
    }

    return (
        <Dialog
            open
            onOpenChange={(open) => {
                if (!open) handleClose();
            }}
        >
            <DialogContent className="sm:max-w-3xl">
                <DialogHeader className="sr-only">
                    <DialogTitle>Bulk Invite Nurses</DialogTitle>
                </DialogHeader>
                <BulkStaffImport role="NURSE" mode="dialog" onClose={handleClose} />
            </DialogContent>
        </Dialog>
    );
}
