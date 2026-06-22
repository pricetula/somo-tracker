/**
 * Intercepted /finance/add — renders as a dialog overlay via the @modal slot.
 */

"use client";

import { useRouter } from "next/navigation";

import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { BulkStaffImport } from "@/features/staff-import";

export default function InterceptedFinanceAdd() {
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
                    <DialogTitle>Invite Finance Staff</DialogTitle>
                </DialogHeader>
                <BulkStaffImport role="FINANCE" mode="dialog" onClose={handleClose} />
            </DialogContent>
        </Dialog>
    );
}
