/**
 * BulkInviteSelector — pick "Manual Entry" or "Import File".
 *
 * Follows the same pattern as StudentsImportSelector.
 */

"use client";

import { FileText, Keyboard } from "lucide-react";
import { Button } from "@/components/ui/button";

interface BulkInviteSelectorProps {
    onSelect: (type: "manual" | "file") => void;
    isDialogVersion?: boolean;
}

export function BulkInviteSelector({ onSelect }: BulkInviteSelectorProps) {
    return (
        <div className="space-y-6">
            <div className="space-y-1">
                <h3 className="text-lg font-semibold">Invite Staff Members</h3>
                <p className="text-muted-foreground text-sm">How would you like to add people?</p>
            </div>

            <div className="grid gap-4 sm:grid-cols-2">
                <Button
                    variant="outline"
                    size="lg"
                    onClick={() => onSelect("manual")}
                    className="flex h-auto flex-col gap-3 p-6"
                >
                    <Keyboard className="size-8" />
                    <div className="space-y-1">
                        <p className="text-base font-medium">Manual Entry</p>
                        <p className="text-muted-foreground text-xs">
                            Type email addresses one by one
                        </p>
                    </div>
                </Button>

                <Button
                    variant="outline"
                    size="lg"
                    onClick={() => onSelect("file")}
                    className="flex h-auto flex-col gap-3 p-6"
                >
                    <FileText className="size-8" />
                    <div className="space-y-1">
                        <p className="text-base font-medium">Import File</p>
                        <p className="text-muted-foreground text-xs">Upload a CSV or Excel file</p>
                    </div>
                </Button>
            </div>
        </div>
    );
}
