"use client";

import { Button } from "@/components/ui/button";
import { UserPlus } from "lucide-react";

export function EmptyState({ onCreateLink }: { onCreateLink: () => void }) {
    return (
        <div className="bg-muted/30 flex items-center justify-center rounded-md px-4 py-8">
            <div className="text-center">
                <p className="text-muted-foreground font-medium">No linked students</p>
                <p className="text-muted-foreground mt-1">
                    Link a student to this parent to manage guardian relationships.
                </p>
                <Button variant="outline" size="sm" className="mt-4" onClick={onCreateLink}>
                    <UserPlus className="mr-1.5 size-3.5" />
                    Link Student
                </Button>
            </div>
        </div>
    );
}
