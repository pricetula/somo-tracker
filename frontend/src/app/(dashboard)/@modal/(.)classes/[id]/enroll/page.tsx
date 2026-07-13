/**
 * Intercepted route — Enrollment overlay as a second-layer side sheet.
 *
 * Slides out on top of the class detail sheet when the user clicks
 * "Enroll Students". Shows a searchable checklist for batch enrollment.
 * On hard refresh the full page at /classes/:id/enroll takes over.
 */

"use client";

import { use } from "react";
import { useRouter } from "next/navigation";
import { EnrollStudentsPanel } from "@/features/classes";
import {
    Sheet,
    SheetContent,
    SheetHeader,
    SheetTitle,
    SheetDescription,
} from "@/components/ui/sheet";

interface Props {
    params: Promise<{ id: string }>;
}

export default function EnrollStudentsOverlay({ params }: Props) {
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
                    <SheetTitle>Enroll Students</SheetTitle>
                    <SheetDescription>
                        Search and select students to enroll in this class.
                    </SheetDescription>
                </SheetHeader>
                <div className="flex-1 overflow-y-auto px-6 pb-6">
                    <EnrollStudentsPanel classId={id} onSuccess={() => router.back()} />
                </div>
            </SheetContent>
        </Sheet>
    );
}
