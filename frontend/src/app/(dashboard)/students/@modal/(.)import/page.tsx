/**
 * Intercepted Import — renders the import pipeline as a dialog overlay.
 *
 * When navigating from /students to /students/import, Next.js intercepts
 * the route and renders this page inside the @modal parallel slot,
 * keeping the students listing mounted underneath.
 */

"use client";

import * as React from "react";
import { useRouter } from "next/navigation";

import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";

import { ImportStoreProvider } from "@/features/students";
import { ImportStageSwitcher } from "@/features/students/components/import-stage-switcher";

export default function InterceptedStudentImportPage() {
    const router = useRouter();

    const handleClose = React.useCallback(() => {
        router.push("/students");
    }, [router]);

    return (
        <Dialog open onOpenChange={(open) => !open && handleClose()}>
            <DialogContent className="flex max-h-[90vh] max-w-4xl flex-col overflow-hidden">
                <DialogHeader>
                    <DialogTitle>Import Students</DialogTitle>
                </DialogHeader>
                <ImportStoreProvider>
                    <ImportStageSwitcher onClose={handleClose} />
                </ImportStoreProvider>
            </DialogContent>
        </Dialog>
    );
}
