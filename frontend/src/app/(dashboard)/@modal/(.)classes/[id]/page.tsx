"use client";

import { useRouter, useParams } from "next/navigation";
import { ClassDetail } from "@/features/classes";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";

export default function ClassDetailModalPage() {
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
            <SheetContent side="right" className="data-[side=right]:data-[side=right]:sm:max-w-2xl">
                <SheetHeader>
                    <SheetTitle>Class Detailsss</SheetTitle>
                </SheetHeader>
                <ClassDetail id={id} />
            </SheetContent>
        </Sheet>
    );
}
