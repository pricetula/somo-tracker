"use client";

import { useRouter } from "next/navigation";
import { FinanceInviteOrchestrator } from "@/features/finance";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";

export default function FinanceInviteModalPage() {
    const router = useRouter();

    const handleOpenChange = (open: boolean) => {
        if (!open) {
            router.back();
        }
    };

    return (
        <Dialog open onOpenChange={handleOpenChange}>
            <DialogContent className="max-h-[85vh] overflow-y-auto md:max-w-2xl">
                <DialogHeader>
                    <DialogTitle>Invite Finances</DialogTitle>
                </DialogHeader>
                <FinanceInviteOrchestrator />
            </DialogContent>
        </Dialog>
    );
}
