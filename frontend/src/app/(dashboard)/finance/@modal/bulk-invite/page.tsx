"use client";

import * as React from "react";
import { BulkInviteForm } from "@/components/shared/bulk-invite/bulk-invite-form";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { useRouter } from "next/navigation";
import { ArrowLeft } from "lucide-react";

export default function FinanceBulkInviteSlot() {
    const router = useRouter();

    const handleClose = () => {
        router.push("/finance");
    };

    return (
        <Dialog
            open
            onOpenChange={(next) => {
                if (!next) handleClose();
            }}
        >
            <DialogContent className="w-full max-w-md">
                <DialogHeader>
                    <DialogTitle>Invite Finance Staff</DialogTitle>
                    <DialogDescription>
                        Upload CSV or manually enter email addresses to invite new finance staff.
                    </DialogDescription>
                </DialogHeader>
                <BulkInviteForm role="FINANCE" />
                <div className="mt-4 flex justify-end">
                    <Button variant="outline" onClick={handleClose}>
                        <ArrowLeft className="mr-2 h-4 w-4" /> Back to Finance
                    </Button>
                </div>
            </DialogContent>
        </Dialog>
    );
}
