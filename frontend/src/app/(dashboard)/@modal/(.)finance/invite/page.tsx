"use client";

import { FinanceInviteOrchestrator } from "@/features/finance";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";

export default function FinanceInviteModalPage() {
    return (
        <Dialog open>
            <DialogContent className="max-h-[85vh] max-w-4xl overflow-y-auto">
                <DialogHeader>
                    <DialogTitle>Invite Finances</DialogTitle>
                </DialogHeader>
                <FinanceInviteOrchestrator />
            </DialogContent>
        </Dialog>
    );
}
