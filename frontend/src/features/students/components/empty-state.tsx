/**
 * Empty State — displayed when zero student records exist.
 *
 * Hyper-minimalist: no clip-art graphics, just centered typography
 * with two secondary-styled CTA buttons.
 */

"use client";

import { Button } from "@/components/ui/button";
import { UserPlus, Upload } from "lucide-react";

interface EmptyStateProps {
    onManualAdd: () => void;
    onUploadCSV: () => void;
}

export function EmptyState({ onManualAdd, onUploadCSV }: EmptyStateProps) {
    return (
        <div className="flex flex-1 items-center justify-center p-8">
            <div className="text-center">
                <p className="text-sm font-medium">No students yet</p>
                <p className="text-muted-foreground mt-1 text-xs">
                    Add your first student to get started with tracking.
                </p>
                <div className="mt-5 flex items-center justify-center gap-3">
                    <Button variant="secondary" size="sm" onClick={onManualAdd}>
                        <UserPlus className="mr-1.5 size-3.5" />
                        Manually Add
                    </Button>
                    <Button variant="secondary" size="sm" onClick={onUploadCSV}>
                        <Upload className="mr-1.5 size-3.5" />
                        Upload CSV
                    </Button>
                </div>
            </div>
        </div>
    );
}
