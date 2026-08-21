/**
 * Parents invitations listing page — shows all sent parent invitations with revoke support.
 *
 * Maps to GET /api/v1/invitations?role=PARENT.
 */

"use client";

import Link from "next/link";
import { Upload } from "lucide-react";

import { Button } from "@/components/ui/button";
import { InvitationsList } from "@/features/invitations";

export default function ParentsInvitationsPage() {
    return (
        <div className="flex flex-1 flex-col gap-4">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-semibold tracking-tight">Parent Invitations</h1>
                    <p className="text-muted-foreground mt-0.5 text-xs">
                        Sent invitations to parent/guardian email addresses.
                    </p>
                </div>
                <Link href="/parents/import">
                    <Upload className="mr-1.5 size-3.5" />
                    Invite Parents
                </Link>
            </div>
            <InvitationsList
                role="PARENT"
                queryKey={["parents", "invitations"]}
                emptyState="No parent invitations sent yet."
            />
        </div>
    );
}
