/**
 * Admins invitations listing page — dedicated view for all sent invitations.
 *
 * TODO: Implement the full listing table with search, filtering, pagination,
 * and the ability to revoke / resend invitations.
 */

"use client";

import * as React from "react";
import Link from "next/link";

import { Button } from "@/components/ui/button";
import { UserPlus } from "lucide-react";

export default function AdminsInvitationsPage() {
    return (
        <div className="flex flex-1 flex-col">
            {/* Page header */}
            <div className="flex items-center gap-3 px-6 pt-6 pb-2">
                <h1 className="text-2xl font-semibold tracking-tight">Admin Invitations</h1>
                <div className="ml-auto">
                    <Button size="sm" asChild>
                        <Link href="/admins/invitations/new">
                            <UserPlus className="mr-1.5 size-3.5" />
                            Bulk Invite
                        </Link>
                    </Button>
                </div>
            </div>

            {/* Placeholder — will be replaced with a dedicated listing */}
            <div className="text-muted-foreground flex flex-1 items-center justify-center px-6">
                <p className="text-sm">Invitation listing coming soon.</p>
            </div>
        </div>
    );
}
