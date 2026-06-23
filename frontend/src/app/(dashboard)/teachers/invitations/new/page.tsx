/**
 * Standalone /teachers/invitations/new — full-page bulk invite form.
 *
 * When accessed directly (hard reload, shared link, new tab), this page
 * renders the BulkStaffImport component in mode='page' with breadcrumb
 * back-link and optional success banner.
 */

"use client";

import * as React from "react";
import Link from "next/link";
import { ChevronLeft } from "lucide-react";

import { BulkStaffImport } from "@/features/staff-import";
import { Badge } from "@/components/ui/badge";

export default function StandaloneTeachersBulkInvite() {
    const [importComplete, setImportComplete] = React.useState(false);

    return (
        <div className="mx-auto flex w-full max-w-4xl flex-col px-6 py-6">
            {/* Breadcrumb back-link */}
            <div className="mb-4">
                <Link
                    href="/teachers/invitations"
                    className="text-muted-foreground hover:text-foreground inline-flex items-center gap-1 text-sm transition-colors"
                >
                    <ChevronLeft className="size-4" />
                    Back to Invitations
                </Link>
            </div>

            {/* Success banner */}
            {importComplete && (
                <div className="mb-4">
                    <Badge
                        variant="default"
                        className="bg-emerald-600 text-xs font-medium text-white hover:bg-emerald-500"
                    >
                        Import completed successfully
                    </Badge>
                    <p className="text-muted-foreground mt-1 text-xs">
                        All invitations have been processed. You can close this page or invite more
                        staff below.
                    </p>
                </div>
            )}

            {/* Page heading */}
            <div className="mb-6">
                <h1 className="text-2xl font-semibold tracking-tight">Bulk Invite Teachers</h1>
                <p className="text-muted-foreground mt-1 text-sm">
                    Invite teachers manually or via file upload.
                </p>
            </div>

            <BulkStaffImport role="TEACHER" mode="page" onSuccess={() => setImportComplete(true)} />
        </div>
    );
}
