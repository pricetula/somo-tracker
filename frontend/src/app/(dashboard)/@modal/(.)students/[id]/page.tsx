/**
 * Intercepted route — Student detail rendered as a sliding side sheet.
 *
 * Mirrors the layout of the full page via the shared StudentDetailContent
 * component, but rendered inside a Sheet so the list stays visible.
 * On hard refresh the full page at /students/[id] takes over.
 */

"use client";

import { use } from "react";
import { useRouter } from "next/navigation";

import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { StudentDetailContent } from "@/features/students";

interface Props {
    params: Promise<{ id: string }>;
}

export default function StudentDetailSheet({ params }: Props) {
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
                    <SheetTitle>Student Profile</SheetTitle>
                </SheetHeader>

                <div className="p-6">
                    <StudentDetailContent
                        studentId={id}
                        variant="sheet"
                        onDeleteSuccess={() => router.back()}
                    />
                </div>
            </SheetContent>
        </Sheet>
    );
}
