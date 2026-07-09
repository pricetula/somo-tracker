/**
 * Parents bulk import page — standalone route.
 *
 * Bulk import UI will be implemented later, following the students import
 * pattern (ImportJob system). For now, shows a placeholder.
 *
 * Maps to POST /api/v1/parents/import.
 */

"use client";

import Link from "next/link";
import { Button } from "@/components/ui/button";
import { ArrowLeft, Upload } from "lucide-react";

export default function ParentsBulkImportPage() {
    return (
        <div className="mx-auto flex max-w-2xl flex-col gap-6 px-6 pt-6 pb-8">
            {/* Back link */}
            <div>
                <Button variant="ghost" size="sm" asChild>
                    <Link href="/parents">
                        <ArrowLeft className="mr-1.5 size-3.5" />
                        Back to Parents
                    </Link>
                </Button>
            </div>

            {/* Header */}
            <div>
                <h1 className="text-2xl font-semibold tracking-tight">Bulk Import Parents</h1>
                <p className="text-muted-foreground mt-1 text-sm">
                    Add multiple parents at once by uploading a CSV file or entering details
                    manually.
                </p>
            </div>

            {/* Placeholder */}
            <div className="border-muted flex flex-col items-center gap-4 rounded-lg border border-dashed px-6 py-16">
                <div className="bg-muted/50 flex size-12 items-center justify-center rounded-full">
                    <Upload className="text-muted-foreground size-5" />
                </div>
                <div className="text-center">
                    <p className="text-sm font-medium">Bulk import coming soon</p>
                    <p className="text-muted-foreground mt-1 text-xs">
                        You&apos;ll be able to import parents via CSV upload or manual entry.
                    </p>
                </div>
                <Button variant="outline" size="sm" asChild>
                    <Link href="/parents/new">Add a single parent instead</Link>
                </Button>
            </div>
        </div>
    );
}
