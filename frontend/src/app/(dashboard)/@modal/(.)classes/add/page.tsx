/**
 * Intercepted route — Add class rendered as a dialog overlay.
 *
 * Slides in as a centered modal when the user clicks "Add Class"
 * from the classes listing page.
 * On hard refresh the full page at /classes/add takes over.
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
import { AddClassForm } from "@/features/classes";

export default function AddClassModal() {
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
                    <DialogTitle>Create Class</DialogTitle>
                    <DialogDescription>
                        Create a new class by selecting a grade level, stream, and academic year.
                    </DialogDescription>
                </DialogHeader>
                <AddClassForm />
            </DialogContent>
        </Dialog>
    );
}
