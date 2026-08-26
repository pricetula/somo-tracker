/**
 * Intercepted route — Create timetable rendered as a dialog overlay.
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
import { CreateTimetable } from "@/features/timetable";

export default function TimetableCreateModal() {
    const router = useRouter();

    const handleRouteBack = useCallback(() => {
        router.back();
    }, [router]);

    return (
        <Dialog open onOpenChange={handleRouteBack}>
            <DialogContent className="max-h-[90vh] w-full overflow-y-auto sm:max-w-2xl">
                <DialogHeader>
                    <DialogTitle>Create Timetable</DialogTitle>
                    <DialogDescription>
                        Set up a new timetable track with time blocks.
                    </DialogDescription>
                </DialogHeader>
                <CreateTimetable handleRouteBack={handleRouteBack} />
            </DialogContent>
        </Dialog>
    );
}
