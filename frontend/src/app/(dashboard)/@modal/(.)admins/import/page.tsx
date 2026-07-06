/**
 * Intercepted route for admin bulk import via the modal slot.
 *
 * Renders an empty dialog overlay — bulk import UI will be implemented later.
 */

"use client";

import { useRouter } from "next/navigation";
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";

export default function AdminsBulkImportModal() {
    const router = useRouter();

    return (
        <Dialog
            open
            onOpenChange={(open) => {
                if (!open) router.back();
            }}
        >
            <DialogContent>
                <DialogHeader>
                    <DialogTitle>Bulk Import Admins</DialogTitle>
                    <DialogDescription>
                        Bulk import functionality will be available soon.
                    </DialogDescription>
                </DialogHeader>
                <div className="text-muted-foreground flex items-center justify-center py-12 text-sm">
                    Coming soon
                </div>
            </DialogContent>
        </Dialog>
    );
}
