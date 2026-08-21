/**
 * Intercepted route — Add class rendered as a dialog overlay.
 *
 * Slides in as a centered modal when the user clicks "Add Class"
 * from the classes listing page.
 * On hard refresh the full page at /classes/add takes over.
 */

"use client";

import React from "react";
import { useRouter } from "next/navigation";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
} from "@/components/ui/dialog";
import { AddClassForm } from "@/features/classes";

export default function AddClassModal() {
    const router = useRouter();

    const handleDialogOpen = React.useCallback(
        (open: boolean) => {
            if (!open) router.back();
        },
        [router]
    );

    return (
        <Dialog open onOpenChange={handleDialogOpen}>
            <DialogContent className="sm:max-w-md">
                <DialogHeader>
                    <DialogTitle>Create Class</DialogTitle>
                    <DialogDescription>
                        Create a new class by selecting a grade level, stream, and academic year.
                    </DialogDescription>
                </DialogHeader>
                <AddClassForm onSuccess={() => handleDialogOpen(false)} />
            </DialogContent>
        </Dialog>
    );
}
