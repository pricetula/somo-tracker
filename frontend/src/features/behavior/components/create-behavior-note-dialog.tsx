"use client";

import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";

interface CreateBehaviorNoteDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    /** Pre-filled timetable slot ID. */
    timetableSlotId: string;
    /** Pre-filled student ID. */
    studentId: string;
    /** Pre-filled date. */
    date: string;
}

import { BehaviorNoteForm } from "./behavior-note-form";

export function CreateBehaviorNoteDialog({
    open,
    onOpenChange,
    timetableSlotId,
    studentId,
    date,
}: CreateBehaviorNoteDialogProps) {
    // Key changes on every new set of pre-filled props, forcing remount
    // which resets all form state without needing useEffect + setState.
    const formKey = `${timetableSlotId}-${studentId}-${date}`;

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent>
                <DialogHeader>
                    <DialogTitle>Log Behavior Note</DialogTitle>
                    <DialogDescription>
                        Record an incident or behavior observation for this student.
                    </DialogDescription>
                </DialogHeader>

                <BehaviorNoteForm
                    key={formKey}
                    timetableSlotId={timetableSlotId}
                    studentId={studentId}
                    date={date}
                    onSubmitSuccess={() => onOpenChange(false)}
                />
            </DialogContent>
        </Dialog>
    );
}
