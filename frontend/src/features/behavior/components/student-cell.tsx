"use client";

import { Badge } from "@/components/ui/badge";
import { AlertTriangle } from "lucide-react";
import { type PendingNoteItem } from "@/lib/api/behavior";

export function StudentCell({ note }: { note: PendingNoteItem }) {
    return (
        <div className="flex items-center gap-2">
            <span className="font-medium">{note.student_full_name}</span>
            <Badge variant="outline" className="text-[10px]">
                {note.class_name}
            </Badge>
            <Badge className="text-[10px]">{note.category_name}</Badge>
            {note.is_urgent && (
                <Badge variant="destructive" className="gap-1 text-[10px]">
                    <AlertTriangle className="h-3 w-3" />
                    Urgent
                </Badge>
            )}
        </div>
    );
}
