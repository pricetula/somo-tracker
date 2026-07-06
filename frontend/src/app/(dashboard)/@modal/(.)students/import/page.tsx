/**
 * Intercepted route for student bulk import via the modal slot.
 *
 * Renders an empty dialog overlay — bulk import UI will be implemented later.
 */

"use client";

import { useRouter } from "next/navigation";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { StudentsImportForm } from "@/features/students/components/students-import/students-import";

export default function StudentsImportModal() {
    const router = useRouter();

    return (
        <Dialog
            open
            onOpenChange={(open) => {
                if (!open) router.back();
            }}
        >
            <DialogContent className="sm:max-w-2xl">
                <DialogHeader>
                    <DialogTitle>Add students</DialogTitle>
                </DialogHeader>
                <StudentsImportForm isDialogVersion={true} />
            </DialogContent>
        </Dialog>
    );
}
