/**
 * Intercepted route — Finance staff detail rendered as a sliding side sheet.
 *
 * When a user clicks a finance staff name in the finance table, this sheet
 * slides out from the right keeping the master table visible but dimmed.
 * On hard refresh the full page at /finance/[id] takes over.
 */

"use client";

import { use } from "react";
import { useRouter } from "next/navigation";
import { FinanceDetail } from "@/features/finance";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";

interface Props {
    params: Promise<{ id: string }>;
}

export default function FinanceDetailSheet({ params }: Props) {
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
                    <SheetTitle>Finance Staff Details</SheetTitle>
                </SheetHeader>
                <div className="flex-1 overflow-y-auto px-6 pb-6">
                    <FinanceDetail id={id} />
                </div>
            </SheetContent>
        </Sheet>
    );
}
