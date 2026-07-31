"use client";

import { type PendingNoteItem } from "@/lib/api/behavior";

export function DescriptionCell({ note }: { note: PendingNoteItem }) {
    return (
        <div className="space-y-1">
            <p className="text-muted-foreground line-clamp-2 text-xs">{note.description}</p>
            <p className="text-muted-foreground text-[10px]">
                By {note.authored_by_name} &middot; {note.date}
            </p>
        </div>
    );
}
