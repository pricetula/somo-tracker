"use client";

import { GuardianInviteOrchestrator } from "@/features/guardians";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";

export default function GuardianInviteModalPage() {
    return (
        <Dialog open>
            <DialogContent className="max-h-[85vh] max-w-4xl overflow-y-auto">
                <DialogHeader>
                    <DialogTitle>Invite Guardians</DialogTitle>
                </DialogHeader>
                <GuardianInviteOrchestrator />
            </DialogContent>
        </Dialog>
    );
}
