"use client";

import { useParams, useRouter } from "next/navigation";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { GuardianDetail } from "@/features/guardians";

export default function GuardianDetailSheet() {
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
            <SheetContent side="right" className="w-full data-[side=right]:sm:max-w-2xl">
                <SheetHeader>
                    <SheetTitle>Guardian</SheetTitle>
                </SheetHeader>
                <GuardianDetail id={id} />
            </SheetContent>
        </Sheet>
    );
}
