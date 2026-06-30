"use client";

import { Switch } from "@/components/ui/switch";
import type { TeacherMember } from "@/lib/api/teachers";

interface TeacherStatusToggleCellProps {
    teacher: TeacherMember;
    onToggle: (userId: string, isActive: boolean) => void;
    isPending: boolean;
}

export function TeacherStatusToggleCell({
    teacher,
    onToggle,
    isPending,
}: TeacherStatusToggleCellProps) {
    const handleToggle = (checked: boolean) => {
        onToggle(teacher.id, checked);
    };

    return (
        <div className="flex items-center gap-2">
            <Switch
                checked={teacher.is_active}
                onCheckedChange={handleToggle}
                disabled={isPending}
                aria-label={teacher.is_active ? "Deactivate teacher" : "Activate teacher"}
            />
            <span
                className={
                    teacher.is_active
                        ? "text-xs font-medium text-emerald-600"
                        : "text-muted-foreground text-xs"
                }
            >
                {teacher.is_active ? "Active" : "Inactive"}
            </span>
        </div>
    );
}
