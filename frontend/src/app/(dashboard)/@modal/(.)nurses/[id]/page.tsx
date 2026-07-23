/**
 * Intercepted route — Nurse detail rendered as a sliding side sheet.
 *
 * When a user clicks a nurse name in the nurses table, this sheet
 * slides out from the right keeping the master table visible but dimmed.
 * On hard refresh the full page at /nurses/[id] takes over.
 */

"use client";

import { use } from "react";
import { useRouter } from "next/navigation";
import { NurseDetail } from "@/features/nurses";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";

interface Props {
    params: Promise<{ id: string }>;
}

export default function NurseDetailSheet({ params }: Props) {
    const router = useRouter();
    const { id } = use(params);

    return (
        <Sheet
            open
            onOpenChange={(open) => {
                if (!open) router.back();
            }}
        >
            <SheetContent side="right" className="w-full data-[side=right]:sm:max-w-xl">
                <SheetHeader>
                    <SheetTitle>Nurse Details</SheetTitle>
                </SheetHeader>
                <div className="flex-1 overflow-y-auto px-6 pb-6">
                    <NurseDetail id={id} />
                </div>
            </SheetContent>
        </Sheet>
    );
}
