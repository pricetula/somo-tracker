"use client";

/**
 * Modal route — renders CreateSchoolForm wrapped in a Dialog shell.
 * Matches the intercepting route `@modal/(.)schools/new`.
 */

import { CreateSchoolForm } from "@/features/school";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";

export default function NewSchoolModal() {
    return (
        <Dialog open>
            <DialogContent className="max-h-[85vh] max-w-xl overflow-y-auto">
                <DialogHeader>
                    <DialogTitle>Create School</DialogTitle>
                </DialogHeader>
                <CreateSchoolForm />
            </DialogContent>
        </Dialog>
    );
}
