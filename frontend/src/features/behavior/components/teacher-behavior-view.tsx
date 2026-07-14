/**
 * TeacherBehaviorView — shows a teacher's own submitted behavior notes.
 */

"use client";

import { Skeleton } from "@/components/ui/skeleton";

// TODO: Fetch teacher's submitted behavior notes
// GET /api/v1/behavior/notes?authored_by_id=:userId

export function TeacherBehaviorView() {
    return (
        <div className="space-y-4">
            <p className="text-muted-foreground text-sm">
                Notes you have submitted. They appear here once reviewed by an admin.
            </p>

            {/* TODO: Replace with actual list */}
            <div className="space-y-3">
                <Skeleton className="h-24 w-full rounded-lg" />
                <Skeleton className="h-24 w-full rounded-lg" />
            </div>
        </div>
    );
}
