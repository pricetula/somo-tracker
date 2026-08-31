/**
 * Intercepted route — Enrollment overlay as a second-layer side sheet.
 *
 * Academic year and term are resolved server-side from the current active term.
 * No user selection is required.
 *
 * Slides out on top of the class detail sheet when the user clicks
 * "Enroll Students". On hard refresh the full page at /classes/:id/enroll takes over.
 */

"use client";

import { Suspense, use } from "react";
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

function EnrollStudentsSheetContent({ classId }: { classId: string }) {
    const router = useRouter();

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
                        Search and select students to enroll in this class. The current academic
                        term will be used automatically.
                    </SheetDescription>
                </SheetHeader>
                <div className="flex-1 overflow-y-auto px-6 pb-6">
                    <EnrollStudentsPanel classId={classId} onSuccess={() => router.back()} />
                </div>
            </SheetContent>
        </Sheet>
    );
}

export default function EnrollStudentsOverlay({ params }: Props) {
    const { id } = use(params);

    return (
        <Suspense
            fallback={
                <Sheet open>
                    <SheetContent side="right" className="w-full data-[side=right]:sm:max-w-xl">
                        <SheetHeader>
                            <SheetTitle>Enroll Students</SheetTitle>
                        </SheetHeader>
                        <div className="flex-1 overflow-y-auto px-6 pb-6">
                            <div className="space-y-4">
                                <div className="bg-muted h-10 w-full animate-pulse rounded" />
                                <div className="bg-muted h-10 w-full animate-pulse rounded" />
                            </div>
                        </div>
                    </SheetContent>
                </Sheet>
            }
        >
            <EnrollStudentsSheetContent classId={id} />
        </Suspense>
    );
}
