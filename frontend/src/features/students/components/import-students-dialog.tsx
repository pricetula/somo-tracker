/**
 * Import Students Dialog — full multi-stage bulk import pipeline.
 *
 * Stages:
 *   1. MAPPING    — Upload CSV/Excel, map columns, select academic term
 *   2. PREVIEW    — Review parsed rows, fix errors, skip rows
 *   3. READY      — Summary and dispatch
 *   4. SUBMITTING — Submit job, watch progress via SSE
 *
 * Resiliency: full state is persisted in IndexedDB. Dialog close / refresh
 * restores the exact stage, column mapping, and in-flight job.
 */

"use client";

import * as React from "react";

import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { ImportStoreProvider } from "../hooks/use-import-store";
import { ImportStageSwitcher } from "./import-stage-switcher";

// ─── Props ─────────────────────────────────────────────────────────────────

interface ImportStudentsDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function ImportStudentsDialog({ open, onOpenChange }: ImportStudentsDialogProps) {
    // Stage is managed inside the store; this outer shell just wraps the stage switcher
    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="flex max-h-[90vh] max-w-4xl flex-col overflow-hidden">
                <DialogHeader>
                    <DialogTitle>Import Students</DialogTitle>
                </DialogHeader>
                <ImportStoreProvider>
                    <ImportStageSwitcher onClose={() => onOpenChange(false)} />
                </ImportStoreProvider>
            </DialogContent>
        </Dialog>
    );
}
