/**
 * Intercepted route for admin bulk invite via the modal slot.
 *
 * Renders BulkInviteForm inside a Dialog with role=SCHOOL_ADMIN.
 */

"use client";

import { useRouter } from "next/navigation";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { BulkInviteForm } from "@/components/shared/bulk-invite";

export default function AdminsBulkImportModal() {
    const router = useRouter();

    return (
        <Dialog
            open
            onOpenChange={(open) => {
                if (!open) router.back();
            }}
        >
            <DialogContent className="sm:max-w-3xl">
                <DialogHeader>
                    <DialogTitle>Invite Admins</DialogTitle>
                </DialogHeader>
                <BulkInviteForm role="SCHOOL_ADMIN" />
            </DialogContent>
        </Dialog>
    );
}
