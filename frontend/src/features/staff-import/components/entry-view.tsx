/**
 * Entry View — the initial tabbed view for the bulk staff import flow.
 *
 * Shows either a "resume draft" prompt or tabs for manual entry / file upload.
 */

"use client";

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { FileUp, UserPlus } from "lucide-react";

import { ManualEntryPanel } from "./manual-entry-panel";
import { FileUploadPanel } from "./file-upload-panel";

import type { ImportDraftRow } from "@/lib/db";
import type { AllowedRole } from "./bulk-staff-import-dialog";

// ─── Types ─────────────────────────────────────────────────────────────────

export interface EntryViewProps {
    hasResumePrompt: boolean;
    onResume: () => void;
    onClear: () => void;
    onRowsReady: (rows: ImportDraftRow[]) => void;
    role: AllowedRole;
    tenantID: string;
    userID: string;
    context: string;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function EntryView({
    hasResumePrompt,
    onResume,
    onClear,
    onRowsReady,
    role,
    tenantID,
    userID,
    context,
}: EntryViewProps) {
    if (hasResumePrompt) {
        return (
            <div className="flex flex-col items-center justify-center gap-4 py-16">
                <p className="text-muted-foreground text-sm">
                    You have an unfinished import draft. Would you like to resume it?
                </p>
                <div className="flex gap-3">
                    <button
                        onClick={onResume}
                        className="bg-primary text-primary-foreground hover:bg-primary/90 rounded-md px-4 py-2 text-sm font-medium"
                    >
                        Resume Draft
                    </button>
                    <button
                        onClick={onClear}
                        className="bg-secondary text-secondary-foreground hover:bg-secondary/80 rounded-md px-4 py-2 text-sm font-medium"
                    >
                        Start Fresh
                    </button>
                </div>
            </div>
        );
    }

    return (
        <Tabs defaultValue="manual" className="flex flex-col">
            <TabsList className="mx-auto mb-4">
                <TabsTrigger value="manual" className="gap-2">
                    <UserPlus className="size-4" />
                    Add Manually
                </TabsTrigger>
                <TabsTrigger value="upload" className="gap-2">
                    <FileUp className="size-4" />
                    Upload File
                </TabsTrigger>
            </TabsList>

            <TabsContent value="manual" className="mt-0 flex-1">
                <ManualEntryPanel
                    onRowsReady={onRowsReady}
                    role={role}
                    tenantID={tenantID}
                    userID={userID}
                    context={context}
                />
            </TabsContent>

            <TabsContent value="upload" className="mt-0 flex-1">
                <FileUploadPanel
                    onRowsReady={onRowsReady}
                    role={role}
                    tenantID={tenantID}
                    userID={userID}
                    context={context}
                />
            </TabsContent>
        </Tabs>
    );
}
