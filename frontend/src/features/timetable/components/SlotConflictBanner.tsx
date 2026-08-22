"use client";

import { AlertCircle, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";

interface Conflict {
    type: "class" | "teacher" | "room";
    message: string;
    structureId: string;
    conflictingSlotId: string;
}

interface SlotConflictBannerProps {
    conflicts: Conflict[];
    onDismiss: (structureId: string) => void;
}

export function SlotConflictBanner({ conflicts, onDismiss }: SlotConflictBannerProps) {
    if (conflicts.length === 0) return null;

    return (
        <div className="space-y-2" role="alert">
            {conflicts.map((conflict, index) => (
                <Alert
                    key={index}
                    variant="destructive"
                    className="border-destructive/30 bg-destructive/5"
                >
                    <AlertCircle className="h-4 w-4" />
                    <div className="flex-1">
                        <AlertTitle>Scheduling Conflict</AlertTitle>
                        <AlertDescription className="text-sm">{conflict.message}</AlertDescription>
                    </div>
                    <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => onDismiss(conflict.structureId)}
                        aria-label="Dismiss conflict"
                    >
                        <X className="h-4 w-4" />
                    </Button>
                </Alert>
            ))}
        </div>
    );
}
