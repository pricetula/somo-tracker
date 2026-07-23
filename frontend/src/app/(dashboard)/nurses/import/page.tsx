/**
 * Nurses bulk invite page.
 *
 * Uses the shared BulkInviteForm with role=NURSE.
 * Invited nurses will have NURSE role in the platform.
 */

"use client";

import { BulkInviteForm } from "@/components/shared/bulk-invite";

export default function NursesBulkImportPage() {
    return (
        <div className="mx-auto flex max-w-2xl flex-col gap-6 px-6 pt-6 pb-8">
            <BulkInviteForm role="NURSE" />
        </div>
    );
}
