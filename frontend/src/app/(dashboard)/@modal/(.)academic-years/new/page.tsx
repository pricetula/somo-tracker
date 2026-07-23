/**
 * Intercepted route for creating a new academic year via the modal slot.
 *
 * Renders the AcademicYearForm inside a Dialog when navigating to
 * /academic-years/new from within the dashboard, preserving the background page.
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
import { AcademicYearForm } from "@/features/academic-years";

export default function NewAcademicYearModal() {
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
                    <DialogTitle>Create Academic Year</DialogTitle>
                    <DialogDescription>
                        Define a new academic year with a name and date range.
                    </DialogDescription>
                </DialogHeader>
                <AcademicYearForm />
            </DialogContent>
        </Dialog>
    );
}
