/**
 * Intercepted /teachers/invitations/new — renders as a dialog overlay via the @modal slot.
 *
 * When users navigate to /teachers/invitations/new from within /teachers (Link click, soft nav),
 * Next.js intercepts the route and renders this page inside the @modal parallel slot,
 * overlaying the /teachers/invitations listing page without unmounting it.
 */

"use client";

import { useRouter } from "next/navigation";

import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { BulkStaffImport } from "@/features/staff-import";

export default function InterceptedTeachersBulkInvite() {
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
                    <DialogTitle>Bulk Invite Teachers</DialogTitle>
                </DialogHeader>
                <BulkStaffImport role="TEACHER" mode="dialog" onClose={handleClose} />
            </DialogContent>
        </Dialog>
    );
}
