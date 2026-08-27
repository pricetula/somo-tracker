/**
 * Create Track — /timetable/new (full page, not intercepted for simplicity)
 */
"use client";

import { TrackCreateForm } from "@/features/timetable/components/track-create-form";

export default function TimetableNewPage() {
    return (
        <div className="mx-auto w-full max-w-lg space-y-6 p-6">
            <div>
                <h1 className="text-2xl font-semibold">New Timetable</h1>
                <p className="text-muted-foreground mt-1">
                    Create a track. Add blocks and assignments after.
                </p>
            </div>
            <TrackCreateForm />
        </div>
    );
}
