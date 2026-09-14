"use client";
/**
 * Modal route — renders AdminInviteOrchestrator wrapped in a Dialog shell.
 * Matches the intercepting route `@modal/(.)admins/invite`.
 */

import { useRouter } from "next/navigation";
import { AdminInviteOrchestrator } from "@/features/admin";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";

export default function ImportModalPage() {
    const router = useRouter();

    const handleOpenChange = (open: boolean) => {
        if (!open) {
            router.back();
        }
    };

    return (
        <Dialog open onOpenChange={handleOpenChange}>
            <DialogContent className="max-h-[85vh] max-w-4xl overflow-y-auto">
                <DialogHeader>
                    <DialogTitle>Invite Users</DialogTitle>
                </DialogHeader>
                <AdminInviteOrchestrator />
            </DialogContent>
        </Dialog>
    );
}
