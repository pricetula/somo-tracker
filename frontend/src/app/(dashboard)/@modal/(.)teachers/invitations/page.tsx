/**
 * Intercepted route — Teacher invitations list rendered as a sliding side sheet.
 *
 * When a user clicks the invitation count badge in the teachers table toolbar,
 * this sheet slides out from the right showing the TEACHER invitations list.
 */

"use client";

import { useRouter } from "next/navigation";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { InvitationsList } from "@/features/invitations";

export default function TeachersInvitationsSheet() {
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
                    <SheetTitle>Teacher Invitations</SheetTitle>
                </SheetHeader>
                <div className="flex-1 overflow-y-auto px-6 pb-6">
                    <InvitationsList
                        role="TEACHER"
                        queryKey={["invitations", "TEACHER"]}
                        emptyState="No teacher invitations yet."
                    />
                </div>
            </SheetContent>
        </Sheet>
    );
}
