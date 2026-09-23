"use client";

/**
 * Modal route — renders CreateSchoolForm wrapped in a Dialog shell.
 * Matches the intercepting route `@modal/(.)schools/new`.
 */

import { useRouter } from "next/navigation";
import { CreateSchoolForm } from "@/features/school";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";

export default function NewSchoolModal() {
    const router = useRouter();

    const handleOpenChange = (open: boolean) => {
        if (!open) {
            router.back();
        }
    };

    return (
        <Dialog open onOpenChange={handleOpenChange}>
            <DialogContent className="max-h-[85vh] overflow-y-auto md:max-w-2xl">
                <DialogHeader>
                    <DialogTitle>Create School</DialogTitle>
                </DialogHeader>
                <CreateSchoolForm />
            </DialogContent>
        </Dialog>
    );
}
