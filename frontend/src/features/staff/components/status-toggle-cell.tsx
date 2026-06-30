"use client";

import { Switch } from "@/components/ui/switch";
import type { Member } from "@/lib/api/admins";

interface StatusToggleCellProps {
    member: Member;
    onToggle: (userId: string, isActive: boolean) => void;
    isPending: boolean;
    label: {
        activate: string;
        deactivate: string;
    };
}

export function StatusToggleCell({ member, onToggle, isPending, label }: StatusToggleCellProps) {
    const handleToggle = (checked: boolean) => {
        onToggle(member.id, checked);
    };

    return (
        <div className="flex items-center gap-2">
            <Switch
                checked={member.is_active}
                onCheckedChange={handleToggle}
                disabled={isPending}
                aria-label={member.is_active ? label.deactivate : label.activate}
            />
            <span
                className={
                    member.is_active
                        ? "text-xs font-medium text-emerald-600"
                        : "text-muted-foreground text-xs"
                }
            >
                {member.is_active ? "Active" : "Inactive"}
            </span>
        </div>
    );
}
