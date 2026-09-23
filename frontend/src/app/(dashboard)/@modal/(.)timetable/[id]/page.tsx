"use client";

import { useParams, useRouter } from "next/navigation";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { TimetableDetail } from "@/features/timetable/components/timetable-detail";

export default function TimetableDetailSheet() {
    const params = useParams();
    const id = params?.id as string;
    const router = useRouter();

    const handleOpenChange = (open: boolean) => {
        if (!open) {
            router.back();
        }
    };

    return (
        <Sheet open onOpenChange={handleOpenChange}>
            <SheetContent side="right" className="w-full sm:max-w-2xl">
                <SheetHeader>
                    <SheetTitle>Timetable</SheetTitle>
                </SheetHeader>
                <TimetableDetail templateId={id} />
            </SheetContent>
        </Sheet>
    );
}
