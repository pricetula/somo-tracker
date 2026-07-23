/**
 * Intercepted route — Batch enrollment overlay as a dialog.
 *
 * Renders on top of the students listing when the user clicks
 * "Enroll Selected Students". On hard refresh, the full page
 * at /students/enroll takes over.
 */

"use client";

import { useRouter } from "next/navigation";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
} from "@/components/ui/dialog";
import { BatchEnrollForm } from "@/features/students";

export default function StudentsEnrollModal() {
    const router = useRouter();

    return (
        <Dialog
            open
            onOpenChange={(open) => {
                if (!open) router.back();
            }}
        >
            <DialogContent className="sm:max-w-md">
                <DialogHeader>
                    <DialogTitle>Enroll Students</DialogTitle>
                    <DialogDescription>
                        Select a class to enroll the selected students in the current academic term.
                    </DialogDescription>
                </DialogHeader>
                <BatchEnrollForm onSuccess={() => router.back()} onCancel={() => router.back()} />
            </DialogContent>
        </Dialog>
    );
}
