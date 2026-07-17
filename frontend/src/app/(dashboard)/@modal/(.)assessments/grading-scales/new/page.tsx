/**
 * Intercepted route — Create scale profile rendered as a dialog overlay.
 *
 * Slides in as a centered modal when the admin clicks "Add Profile"
 * from the grading profiles listing page.
 * On hard refresh the full page at /assessments/grading-scales/new takes over.
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
import { CreateScaleProfileForm } from "@/features/assessments";

export default function CreateScaleProfileModal() {
    const router = useRouter();

    return (
        <Dialog
            open
            onOpenChange={(open) => {
                if (!open) router.back();
            }}
        >
            <DialogContent className="sm:max-w-lg">
                <DialogHeader>
                    <DialogTitle>Create Scale Profile</DialogTitle>
                    <DialogDescription>
                        Define a new set of percentage-to-CBC-level conversion rules.
                    </DialogDescription>
                </DialogHeader>
                <CreateScaleProfileForm />
            </DialogContent>
        </Dialog>
    );
}
