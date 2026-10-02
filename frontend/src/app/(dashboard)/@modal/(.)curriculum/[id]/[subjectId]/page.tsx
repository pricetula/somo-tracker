"use client";

import { useParams, useRouter } from "next/navigation";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { SubjectDetail } from "@/features/curriculum";

export default function SubjectDetailSheet() {
    const params = useParams();
    const router = useRouter();

    const handleOpenChange = (open: boolean) => {
        if (!open) {
            router.back();
        }
    };

    return (
        <Sheet open onOpenChange={handleOpenChange}>
            <SheetContent side="right" className="w-full sm:max-w-md">
                <SheetHeader>
                    <SheetTitle>Subject</SheetTitle>
                </SheetHeader>
                <SubjectDetail
                    gradeId={params?.id as string}
                    subjectId={params?.subjectId as string}
                />
            </SheetContent>
        </Sheet>
    );
}
