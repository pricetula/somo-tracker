/**
 * Intercepted route for student bulk import via the modal slot.
 *
 * Renders an empty dialog overlay — bulk import UI will be implemented later.
 */

"use client";

import React from "react";
import { useRouter } from "next/navigation";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { StudentsImportForm } from "@/features/students/components/students-import/students-import";

export default function StudentsImportModal() {
    const router = useRouter();

    const handleRouteBack = React.useCallback(
        (open: boolean) => {
            if (!open) router.back();
        },
        [router]
    );

    return (
        <Dialog open onOpenChange={handleRouteBack}>
            <DialogContent className="sm:max-w-3xl">
                <DialogHeader>
                    <DialogTitle>Add Students</DialogTitle>
                </DialogHeader>
                <StudentsImportForm
                    isDialogVersion={true}
                    onSuccess={() => handleRouteBack(false)}
                />
            </DialogContent>
        </Dialog>
    );
}
