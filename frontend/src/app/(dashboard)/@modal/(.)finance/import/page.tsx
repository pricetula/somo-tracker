/**
 * Intercepted route for finance staff bulk invite via the modal slot.
 *
 * Renders BulkInviteForm inside a Dialog with role=FINANCE.
 */

"use client";

import { useRouter } from "next/navigation";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { BulkInviteForm } from "@/features/staff/components/bulk-invite";

export default function FinanceBulkImportModal() {
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
                    <DialogTitle>Invite Finance Staff</DialogTitle>
                </DialogHeader>
                <BulkInviteForm role="FINANCE" />
            </DialogContent>
        </Dialog>
    );
}
