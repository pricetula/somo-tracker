/**
 * Intercepted route — Admin invitations list rendered as a sliding side sheet.
 *
 * When a user clicks the invitation count badge in the admins table toolbar,
 * this sheet slides out from the right showing the SCHOOL_ADMIN invitations
 * list while keeping the admins table visible but dimmed.
 */

"use client";

import { useRouter } from "next/navigation";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { InvitationsList } from "@/features/invitations";

export default function AdminsInvitationsSheet() {
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
                    <SheetTitle>Admin Invitations</SheetTitle>
                </SheetHeader>
                <div className="flex-1 overflow-y-auto px-6 pb-6">
                    <InvitationsList
                        role="SCHOOL_ADMIN"
                        queryKey={["invitations", "SCHOOL_ADMIN"]}
                        emptyState="No admin invitations yet."
                    />
                </div>
            </SheetContent>
        </Sheet>
    );
}
