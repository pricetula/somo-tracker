/**
 * Invite Form Dialog Content — the dialog wrapper for the invite form.
 */

"use client";

import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";
import { InviteFormContent } from "./invite-form-content";

// ─── Types ─────────────────────────────────────────────────────────────────

export interface InviteFormDialogContentProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    onSuccess: () => void;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function InviteFormDialogContent({
    open,
    onOpenChange,
    onSuccess,
}: InviteFormDialogContentProps) {
    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="sm:max-w-3xl">
                <DialogHeader>
                    <DialogTitle>Invite Users</DialogTitle>
                    <DialogDescription>
                        Send invitation emails to join your school. You can choose a role for each
                        person.
                    </DialogDescription>
                </DialogHeader>
                <InviteFormContent onSuccess={onSuccess} />
            </DialogContent>
        </Dialog>
    );
}
