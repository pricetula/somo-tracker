/**
 * Nurses bulk invite page.
 *
 * Uses the shared BulkInviteForm with role=NURSE.
 * Invited nurses will have NURSE role in the platform.
 */

"use client";

import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { BulkInviteForm } from "@/features/staff/components/bulk-invite";

export default function NursesBulkImportPage() {
    return (
        <div className="mx-auto flex max-w-2xl flex-col gap-6 px-6 pt-6 pb-8">
            <div>
                <Button variant="ghost" size="sm" asChild>
                    <Link href="/nurses">
                        <ArrowLeft className="mr-1.5 size-3.5" />
                        Back to Nurses
                    </Link>
                </Button>
            </div>

            <BulkInviteForm role="NURSE" />
        </div>
    );
}
