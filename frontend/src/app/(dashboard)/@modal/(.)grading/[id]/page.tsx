/**
 * Intercepted route — Grading profile detail rendered as a side sheet.
 *
 * When the admin clicks a profile name, this sheet slides out from the right.
 * On hard refresh the full page at /grading/[id] takes over.
 */

"use client";

import { use } from "react";
import { useRouter } from "next/navigation";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { ScaleProfileDetailView } from "@/features/assessments";

interface Props {
    params: Promise<{ id: string }>;
}

export default function ScaleProfileDetailSheet({ params }: Props) {
    const router = useRouter();
    const { id } = use(params);

    return (
        <Sheet
            open
            onOpenChange={(open) => {
                if (!open) router.back();
            }}
        >
            <SheetContent
                side="right"
                className="w-full overflow-y-auto data-[side=right]:sm:max-w-xl"
            >
                <SheetHeader>
                    <SheetTitle>Scale Profile</SheetTitle>
                </SheetHeader>
                <div className="flex-1 overflow-y-auto px-6 pt-4 pb-6">
                    <ScaleProfileDetailView profileId={id} />
                </div>
            </SheetContent>
        </Sheet>
    );
}
