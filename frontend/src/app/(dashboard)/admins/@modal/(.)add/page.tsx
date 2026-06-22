/**
 * Intercepted /admins/add — renders as a dialog overlay via the @modal slot.
 *
 * When users navigate to /admins/add from within /admins (Link click, soft nav),
 * Next.js intercepts the route and renders this page inside the @modal parallel
 * slot, overlaying the /admins listing page without unmounting it.
 */

"use client";

import { useRouter } from "next/navigation";

import { Dialog, DialogContent } from "@/components/ui/dialog";
import { BulkStaffImport } from "@/features/staff-import";

export default function InterceptedAdminsAdd() {
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
            <DialogContent className="max-w-4xl">
                <BulkStaffImport role="SCHOOL_ADMIN" mode="dialog" onClose={handleClose} />
            </DialogContent>
        </Dialog>
    );
}
