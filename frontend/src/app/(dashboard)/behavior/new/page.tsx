/**
 * New behavior note page — quick-add form.
 * Pre-filled via query params (timetable_slot_id, student_id, date).
 */

import { Suspense } from "react";
import { CreateBehaviorNoteForm } from "@/features/behavior/components/create-behavior-note-form";

export default function NewBehaviorNotePage() {
    return (
        <div className="space-y-6">
            <h1 className="text-2xl font-bold">New Behavior Note</h1>
            <Suspense
                fallback={<div className="text-muted-foreground py-8 text-center">Loading...</div>}
            >
                <CreateBehaviorNoteForm />
            </Suspense>
        </div>
    );
}
