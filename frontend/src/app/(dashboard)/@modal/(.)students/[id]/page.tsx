/**
 * Intercepted route — Student detail rendered as a sliding side sheet.
 *
 * When a user clicks a student name in the class roster table, this sheet
 * slides out from the right keeping the roster table visible but dimmed.
 * On hard refresh the full page at /students/[id] takes over.
 */

"use client";

import { use } from "react";
import { useRouter } from "next/navigation";
import { Construction } from "lucide-react";

import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";

interface Props {
    params: Promise<{ id: string }>;
}

export default function StudentDetailSheet({ params }: Props) {
    const router = useRouter();
    use(params); // resolve route params

    return (
        <Sheet
            open
            onOpenChange={(open) => {
                if (!open) router.back();
            }}
        >
            <SheetContent side="right" className="w-full data-[side=right]:sm:max-w-xl">
                <SheetHeader>
                    <SheetTitle>Student Profile</SheetTitle>
                </SheetHeader>
                <div className="flex flex-1 flex-col items-center justify-center px-6 py-24">
                    <Construction className="text-muted-foreground mb-4 h-12 w-12" />
                    <p className="text-muted-foreground text-center">
                        The student detail page is coming soon.
                    </p>
                </div>
            </SheetContent>
        </Sheet>
    );
}
