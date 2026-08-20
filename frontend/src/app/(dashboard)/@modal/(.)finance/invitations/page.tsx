/**
 * Intercepted route — Finance invitations list rendered as a sliding side sheet.
 *
 * When a user clicks the invitation count badge in the finance table toolbar,
 * this sheet slides out from the right showing the FINANCE invitations list.
 */

"use client";

import { useRouter } from "next/navigation";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { InvitationsList } from "@/features/invitations";

export default function FinanceInvitationsSheet() {
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
                    <SheetTitle>Finance Invitations</SheetTitle>
                </SheetHeader>
                <div className="flex-1 overflow-y-auto px-6 pb-6">
                    <InvitationsList
                        role="FINANCE"
                        queryKey={["invitations", "FINANCE"]}
                        emptyState="No finance invitations yet."
                    />
                </div>
            </SheetContent>
        </Sheet>
    );
}
