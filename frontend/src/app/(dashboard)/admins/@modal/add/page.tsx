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

export default function AdminsAddSlot() {
    const router = useRouter();

    const handleClose = () => {
        router.push("/admins");
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
                    <DialogTitle>Invite Admins</DialogTitle>
                    <DialogDescription>
                        Upload CSV or manually enter email addresses to invite new admins.
                    </DialogDescription>
                </DialogHeader>
                <BulkInviteForm role="SCHOOL_ADMIN" />
                <div className="mt-4 flex justify-end">
                    <Button variant="outline" onClick={handleClose}>
                        <ArrowLeft className="mr-2 h-4 w-4" /> Back to Admins
                    </Button>
                </div>
            </DialogContent>
        </Dialog>
    );
}
