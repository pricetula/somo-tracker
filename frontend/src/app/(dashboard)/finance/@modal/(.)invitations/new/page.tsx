/**
 * Intercepted /finance/invitations/new — renders as a dialog overlay via the @modal slot.
 *
 * When users navigate to /finance/invitations/new from within /finance (Link click, soft nav),
 * Next.js intercepts the route and renders this page inside the @modal parallel slot,
 * overlaying the /finance/invitations listing page without unmounting it.
 *
 * Mirrors the pattern established by /finance/@modal/(.)add/page.tsx.
 */

"use client";

import { useRouter } from "next/navigation";

import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { BulkStaffImport } from "@/features/staff-import";

export default function InterceptedFinanceBulkInvite() {
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
                    <DialogTitle>Bulk Invite Finance Staff</DialogTitle>
                </DialogHeader>
                <BulkStaffImport role="FINANCE" mode="dialog" onClose={handleClose} />
            </DialogContent>
        </Dialog>
    );
}
