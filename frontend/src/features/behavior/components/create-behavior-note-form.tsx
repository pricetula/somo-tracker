/**
 * CreateBehaviorNoteForm — client component that reads query params and
 * renders the behavior note creation dialog.
 */

"use client";

import { useState } from "react";
import { useSearchParams } from "next/navigation";
import { CreateBehaviorNoteDialog } from "./create-behavior-note-dialog";

export function CreateBehaviorNoteForm() {
    const searchParams = useSearchParams();
    const [open, setOpen] = useState(true);

    const timetableSlotId = searchParams.get("timetable_slot_id") ?? "";
    const studentId = searchParams.get("student_id") ?? "";
    const date = searchParams.get("date") ?? new Date().toISOString().split("T")[0];

    return (
        <CreateBehaviorNoteDialog
            open={open}
            onOpenChange={setOpen}
            timetableSlotId={timetableSlotId}
            studentId={studentId}
            date={date}
        />
    );
}
