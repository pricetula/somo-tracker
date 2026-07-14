/**
 * BehaviorNoteDetail — full detail view of a single behavior note with review actions.
 */

"use client";

import { useParams } from "next/navigation";
import { Skeleton } from "@/components/ui/skeleton";

// TODO: Fetch note by ID: GET /api/v1/behavior/notes/:id

export function BehaviorNoteDetail() {
    const params = useParams();
    void params; // unused until API integration

    return (
        <div className="space-y-6">
            <h1 className="text-2xl font-bold">Behavior Note</h1>

            {/* TODO: Show note details from API */}
            <div className="space-y-3">
                <Skeleton className="h-6 w-48" />
                <Skeleton className="h-20 w-full rounded-lg" />
                <Skeleton className="h-10 w-32" />
            </div>
        </div>
    );
}
