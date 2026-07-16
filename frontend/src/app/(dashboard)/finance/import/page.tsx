/**
 * Finance bulk invite page.
 *
 * Uses the shared BulkInviteForm with role=FINANCE.
 * Invited finance users will have FINANCE role in the platform.
 */

"use client";

import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { BulkInviteForm } from "@/components/shared/bulk-invite";

export default function FinanceBulkImportPage() {
    return (
        <div className="mx-auto flex max-w-2xl flex-col gap-6 px-6 pt-6 pb-8">
            <div>
                <Button variant="ghost" size="sm" asChild>
                    <Link href="/finance">
                        <ArrowLeft className="mr-1.5 size-3.5" />
                        Back to Finance
                    </Link>
                </Button>
            </div>

            <BulkInviteForm role="FINANCE" />
        </div>
    );
}
