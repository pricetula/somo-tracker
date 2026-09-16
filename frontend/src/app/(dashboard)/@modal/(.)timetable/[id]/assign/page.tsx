"use client";

import { useRouter, useParams, useSearchParams } from "next/navigation";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { AssignSlotForm } from "@/features/timetable/components/assign-slot-form";

export default function AssignSlotModalPage() {
    const router = useRouter();
    const params = useParams();
    const searchParams = useSearchParams();

    const id = params.id as string;
    const dayParam = searchParams.get("day");
    const slotParam = searchParams.get("slot");

    const dayOfWeek = Number(dayParam ?? 1);
    const timeSlotId = slotParam ?? "";

    const handleOpenChange = (open: boolean) => {
        if (!open) {
            router.back();
        }
    };

    if (!timeSlotId) {
        // Invalid state – close
        router.back();
        return null;
    }

    return (
        <Dialog open onOpenChange={handleOpenChange}>
            <DialogContent className="max-h-[85vh] max-w-lg overflow-y-auto">
                <DialogHeader>
                    <DialogTitle>Assign Timetable Slot</DialogTitle>
                </DialogHeader>
                <AssignSlotForm
                    dayOfWeek={dayOfWeek}
                    timeSlotId={timeSlotId}
                    onSuccess={() => router.back()}
                />
            </DialogContent>
        </Dialog>
    );
}
