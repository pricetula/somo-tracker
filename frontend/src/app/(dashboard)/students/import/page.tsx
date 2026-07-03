/**
 * Student Import page — full-page bulk import pipeline.
 *
 * Maps to /students/import.
 * When navigated to from within /students, the @modal parallel slot
 * intercepts this route and renders as a dialog overlay instead.
 */

"use client";

import * as React from "react";
import { useRouter } from "next/navigation";

import { Button } from "@/components/ui/button";
import { ArrowLeft } from "lucide-react";

import { ImportStoreProvider } from "@/features/students";
import { ImportStageSwitcher } from "@/features/students/components/import-stage-switcher";

export default function StudentImportPage() {
    const router = useRouter();

    return (
        <div className="mx-auto flex max-w-4xl flex-col px-6 pt-6 pb-8">
            <Button
                variant="ghost"
                size="sm"
                className="mb-4 w-fit"
                onClick={() => router.push("/students")}
            >
                <ArrowLeft className="mr-1.5 size-4" />
                Back to Students
            </Button>

            <div className="mb-6">
                <h1 className="text-2xl font-semibold tracking-tight">Import Students</h1>
                <p className="text-muted-foreground mt-1 text-sm">
                    Upload a CSV or Excel file to bulk-import student records.
                </p>
            </div>

            <ImportStoreProvider>
                <ImportStageSwitcher onClose={() => router.push("/students")} />
            </ImportStoreProvider>
        </div>
    );
}
