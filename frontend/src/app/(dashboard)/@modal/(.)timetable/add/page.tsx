"use client";

import { useRouter } from "next/navigation";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { TimetableTemplateWizard } from "@/features/timetable/components/timetable-template-wizard";

export default function TimetableAddSheet() {
    const router = useRouter();

    const handleOpenChange = (open: boolean) => {
        if (!open) {
            router.back();
        }
    };

    return (
        <Sheet open onOpenChange={handleOpenChange}>
            <SheetContent
                side="right"
                className="w-full overflow-y-auto data-[side=right]:sm:max-w-2xl"
            >
                <SheetHeader>
                    <SheetTitle>Create Timetable Template</SheetTitle>
                </SheetHeader>
                <div className="p-6">
                    <TimetableTemplateWizard />
                </div>
            </SheetContent>
        </Sheet>
    );
}
