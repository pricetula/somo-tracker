/**
 * Parents bulk invite page — standalone route.
 *
 * Uses the shared BulkInviteForm with role=PARENT and the parent-specific
 * submit function that posts to /api/v1/parents/invite.
 */

"use client";

import { BulkInviteForm } from "@/components/shared/bulk-invite";
import { submitParentBulkInvite } from "@/lib/api/parents";

export default function ParentsBulkImportPage() {
    return (
        <div className="mx-auto flex max-w-2xl flex-col gap-6 px-6 pt-6 pb-8">
            <BulkInviteForm role="PARENT" submitFn={submitParentBulkInvite} />
        </div>
    );
}
