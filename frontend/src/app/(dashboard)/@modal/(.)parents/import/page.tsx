/**
 * Intercepted route for parent bulk import via the modal slot.
 *
 * Renders the bulk invite form inside a dialog overlay.
 * Matches the pattern used by admins: /admins/import as a modal.
 */

"use client";

import { useRouter } from "next/navigation";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { BulkInviteForm } from "@/components/shared/bulk-invite";
import { submitParentBulkInvite } from "@/lib/api/parents";

export default function ParentsImportModal() {
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
                    <DialogTitle>Invite Parents</DialogTitle>
                </DialogHeader>
                <BulkInviteForm role="PARENT" submitFn={submitParentBulkInvite} />
            </DialogContent>
        </Dialog>
    );
}
