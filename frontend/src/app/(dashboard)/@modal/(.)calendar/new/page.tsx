"use client";

import { useRouter } from "next/navigation";
import { Dialog, DialogContent, DialogTitle, DialogDescription } from "@/components/ui/dialog";
import { AcademicCalendarForm } from "@/features/calendar";

export default function CalendarNewModal() {
    const router = useRouter();

    return (
        <Dialog
            open
            onOpenChange={(open) => {
                if (!open) router.back();
            }}
        >
            <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-2xl" showCloseButton>
                <DialogTitle className="sr-only">Set Up Academic Calendar</DialogTitle>
                <DialogDescription className="sr-only">
                    Define your academic year periods
                </DialogDescription>
                <AcademicCalendarForm onSuccess={() => router.back()} />
            </DialogContent>
        </Dialog>
    );
}
