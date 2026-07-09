/**
 * Intercepted route for teacher bulk invite via the modal slot.
 *
 * Renders BulkInviteForm inside a Dialog with role=TEACHER.
 */

"use client";

import { useRouter } from "next/navigation";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { BulkInviteForm } from "@/features/staff/components/bulk-invite";

export default function TeachersImportModal() {
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
                    <DialogTitle>Invite Teachers</DialogTitle>
                </DialogHeader>
                <BulkInviteForm role="TEACHER" isDialogVersion />
            </DialogContent>
        </Dialog>
    );
}
