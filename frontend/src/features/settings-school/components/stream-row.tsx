"use client";

import { type Stream } from "@/features/streams";

import { ColorDot } from "./color-dot";
import { RenameStreamDialog } from "./rename-stream-dialog";
import { DeleteStreamAlert } from "./delete-stream-alert";

export function StreamRow({ stream }: { stream: Stream }) {
    return (
        <div className="hover:bg-muted/50 flex items-center justify-between gap-4 rounded-md px-3 py-2">
            <div className="flex items-center gap-3">
                <ColorDot color={stream.color} />
                <span className="text-foreground font-medium">{stream.name}</span>
            </div>
            <div className="flex items-center gap-1">
                <RenameStreamDialog stream={stream} />
                <DeleteStreamAlert stream={stream} />
            </div>
        </div>
    );
}
