/**
 * Teachers bulk invite page.
 *
 * Uses the shared BulkInviteForm with role=TEACHER.
 * Invited teachers will have TEACHER role in the platform.
 */

"use client";

import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { BulkInviteForm } from "@/features/staff/components/bulk-invite";

export default function TeachersImportPage() {
    return (
        <div className="mx-auto flex max-w-2xl flex-col gap-6 px-6 pt-6 pb-8">
            <div>
                <Button variant="ghost" size="sm" asChild>
                    <Link href="/teachers">
                        <ArrowLeft className="mr-1.5 size-3.5" />
                        Back to Teachers
                    </Link>
                </Button>
            </div>

            <BulkInviteForm role="TEACHER" />
        </div>
    );
}
