/**
 * Empty State — displayed when zero member records exist.
 */

"use client";

import { Button } from "@/components/ui/button";
import { Mail } from "lucide-react";

interface EmptyStateProps {
    roleLabel: string;
    onInvite: () => void;
}

export function EmptyState({ roleLabel, onInvite }: EmptyStateProps) {
    return (
        <div className="flex flex-1 items-center justify-center p-8">
            <div className="text-center">
                <p className="text-sm font-medium">No {roleLabel.toLowerCase()} yet</p>
                <p className="text-muted-foreground mt-1 text-xs">
                    Invite {roleLabel.toLowerCase()} to join your school.
                </p>
                <div className="mt-5">
                    <Button variant="secondary" size="sm" onClick={onInvite}>
                        <Mail className="mr-1.5 size-3.5" />
                        Invite {roleLabel}
                    </Button>
                </div>
            </div>
        </div>
    );
}
