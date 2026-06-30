/**
 * Admins listing page — active school administrators.
 *
 * NOTE: The backend does not currently expose an endpoint to list active
 * SCHOOL_ADMIN members (GET /api/v1/members only supports TEACHER, NURSE,
 * FINANCE). This page shows a placeholder until the backend adds support.
 *
 * Invitations are listed on the dedicated /admins/invitations page.
 */

"use client";

import Link from "next/link";

import { Button } from "@/components/ui/button";
import { UserPlus } from "lucide-react";

export default function AdminsPage() {
    return (
        <div className="flex flex-1 flex-col">
            {/* Page header */}
            <div className="flex items-center gap-3 px-6 pt-6 pb-2">
                <h1 className="text-2xl font-semibold tracking-tight">Admins</h1>
                <div className="ml-auto">
                    <Button size="sm" asChild>
                        <Link href="/admins/invitations">
                            <UserPlus className="mr-1.5 size-3.5" />
                            Invite Admins
                        </Link>
                    </Button>
                </div>
            </div>

            <div className="flex flex-1 flex-col px-6 py-4">
                <section className="flex flex-col">
                    <div className="ring-foreground/10 flex items-center justify-center rounded-lg py-16 ring-1">
                        <div className="text-center">
                            <p className="text-muted-foreground text-sm font-medium">
                                Admin listing coming soon
                            </p>
                            <p className="text-muted-foreground mt-1 text-xs">
                                Active school administrators will be listed here once the backend
                                endpoint is available.
                            </p>
                        </div>
                    </div>
                </section>
            </div>
        </div>
    );
}
