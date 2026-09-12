"use client";
/**
 * Modal route — renders AdminInviteOrchestrator wrapped in a Dialog shell.
 * Matches the intercepting route `@modal/(.)admins/invite`.
 */

import { AdminInviteOrchestrator } from "@/features/admin";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";

export default function ImportModalPage() {
    return (
        <Dialog open>
            <DialogContent className="max-h-[85vh] max-w-4xl overflow-y-auto">
                <DialogHeader>
                    <DialogTitle>Invite Users</DialogTitle>
                </DialogHeader>
                <AdminInviteOrchestrator />
            </DialogContent>
        </Dialog>
    );
}
