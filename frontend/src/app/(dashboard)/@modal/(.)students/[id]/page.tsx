"use client";

import { useParams, useRouter } from "next/navigation";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { StudentDetail } from "@/features/students";

export default function StudentDetailSheet() {
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
            <SheetContent side="right" className="w-full sm:max-w-md">
                <SheetHeader>
                    <SheetTitle>Student</SheetTitle>
                </SheetHeader>
                <StudentDetail id={id} />
            </SheetContent>
        </Sheet>
    );
}
