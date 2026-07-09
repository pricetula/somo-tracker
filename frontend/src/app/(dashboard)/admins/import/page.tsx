/**
 * Admins bulk invite page.
 *
 * Uses the shared BulkInviteForm with role=SCHOOL_ADMIN.
 * Invited admins will have SCHOOL_ADMIN role in the platform.
 */

"use client";

import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { BulkInviteForm } from "@/features/staff/components/bulk-invite";

export default function AdminsBulkImportPage() {
    return (
        <div className="mx-auto flex max-w-2xl flex-col gap-6 px-6 pt-6 pb-8">
            <div>
                <Button variant="ghost" size="sm" asChild>
                    <Link href="/admins">
                        <ArrowLeft className="mr-1.5 size-3.5" />
                        Back to Admins
                    </Link>
                </Button>
            </div>

            <BulkInviteForm role="SCHOOL_ADMIN" />
        </div>
    );
}
