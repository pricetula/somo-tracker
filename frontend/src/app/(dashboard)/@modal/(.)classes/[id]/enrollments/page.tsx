"use client";

import { useRouter, useParams } from "next/navigation";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { EnrollmentsPageContent } from "@/features/enrollments";

export default function EnrollmentsModalPage() {
    const router = useRouter();
    const params = useParams();
    const id = params.id as string;

    const handleOpenChange = (open: boolean) => {
        if (!open) {
            router.back();
        }
    };

    return (
        <Sheet open onOpenChange={handleOpenChange}>
            <SheetContent side="right" className="w-full overflow-y-auto sm:max-w-xl">
                <SheetHeader>
                    <SheetTitle>Enroll Students</SheetTitle>
                </SheetHeader>
                <div className="mt-4">
                    <EnrollmentsPageContent classId={id} />
                </div>
            </SheetContent>
        </Sheet>
    );
}
