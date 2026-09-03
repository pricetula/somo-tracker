/**
 * Intercepted allocation — renders as modal overlay when navigating from track detail.
 */
"use client";
import { useCallback } from "react";
import { useRouter } from "next/navigation";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
} from "@/components/ui/dialog";
import { AllocationForm } from "@/features/timetable/components/allocation-form";

export default function AllocateModal() {
    const router = useRouter();
    const handleBack = useCallback(() => router.back(), [router]);

    return (
        <Dialog open onOpenChange={handleBack}>
            <DialogContent className="max-h-[90vh] w-full overflow-y-auto sm:max-w-lg">
                <DialogHeader>
                    <DialogTitle>Assign Teacher</DialogTitle>
                    <DialogDescription>
                        Select teacher and subject for this timetable slot.
                    </DialogDescription>
                </DialogHeader>
                <AllocationForm onSuccess={handleBack} />
            </DialogContent>
        </Dialog>
    );
}
