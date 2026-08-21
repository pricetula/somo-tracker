/**
 * Create School Dialog — modal dialog wrapper for the create-school form.
 *
 * Used both as a standalone controlled dialog and via the intercepted
 * route at /schools/new.
 */

"use client";

import * as React from "react";

import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
} from "@/components/ui/dialog";
import { CreateSchoolForm } from "./create-school-form";

// ─── Props ─────────────────────────────────────────────────────────────────

interface CreateSchoolDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function CreateSchoolDialog({ open, onOpenChange }: CreateSchoolDialogProps) {
    const handleSuccess = () => {
        onOpenChange(false);
    };

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="sm:max-w-md">
                <DialogHeader>
                    <DialogTitle>Add School</DialogTitle>
                    <DialogDescription>
                        Create a new school under your tenant. The CBC curriculum will be
                        automatically seeded.
                    </DialogDescription>
                </DialogHeader>
                <CreateSchoolForm onSuccess={handleSuccess} />
            </DialogContent>
        </Dialog>
    );
}
