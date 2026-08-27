/**
 * Intercepted route — track creation rendered as a dialog overlay.
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
import { TrackCreateForm } from "@/features/timetable/components/track-create-form";

export default function TrackCreateModal() {
    const router = useRouter();
    const handleBack = useCallback(() => router.back(), [router]);

    return (
        <Dialog open onOpenChange={handleBack}>
            <DialogContent className="max-h-[90vh] w-full overflow-y-auto sm:max-w-lg">
                <DialogHeader>
                    <DialogTitle>New Timetable</DialogTitle>
                    <DialogDescription>
                        Create a track. Add blocks and assignments after.
                    </DialogDescription>
                </DialogHeader>
                <TrackCreateForm onSuccess={handleBack} />
            </DialogContent>
        </Dialog>
    );
}
