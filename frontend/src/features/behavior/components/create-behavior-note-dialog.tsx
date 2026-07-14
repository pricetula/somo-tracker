/**
 * CreateBehaviorNoteDialog — quick-add form for logging a behavior note.
 *
 * Opens as a dialog with category select, description textarea, and urgent toggle.
 * Pessimistic save with spinner; dialog stays open on failure to preserve input.
 *
 * Form state resets on each open via a key derived from the pre-filled props,
 * avoiding setState inside useEffect (per frontend AGENTS.md policy).
 */

"use client";

import { useState } from "react";
import { Loader2 } from "lucide-react";

import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";

import { useBehaviorCategories, useCreateBehaviorNote } from "../hooks/use-behavior";

interface CreateBehaviorNoteDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    /** Pre-filled timetable slot ID (from attendance roster). */
    timetableSlotId: string;
    /** Pre-filled student ID. */
    studentId: string;
    /** Pre-filled date. */
    date: string;
}

/** Inner form component — key changes force remount, resetting all state. */
function BehaviorNoteForm({
    timetableSlotId,
    studentId,
    date,
    onSubmitSuccess,
}: {
    timetableSlotId: string;
    studentId: string;
    date: string;
    onSubmitSuccess: () => void;
}) {
    const { data: categoriesData } = useBehaviorCategories(true);
    const createNote = useCreateBehaviorNote();

    const [categoryId, setCategoryId] = useState("");
    const [description, setDescription] = useState("");
    const [isUrgent, setIsUrgent] = useState(false);

    const categories = categoriesData?.items ?? [];

    const handleSubmit = () => {
        if (!categoryId || !description.trim()) return;

        createNote.mutate(
            {
                timetable_slot_id: timetableSlotId,
                student_id: studentId,
                date,
                category_id: categoryId,
                description: description.trim(),
                is_urgent: isUrgent,
            },
            {
                onSuccess: () => {
                    onSubmitSuccess();
                },
            }
        );
    };

    return (
        <>
            <div className="space-y-4">
                <div className="space-y-2">
                    <Label htmlFor="category">Category</Label>
                    <Select value={categoryId} onValueChange={setCategoryId}>
                        <SelectTrigger id="category">
                            <SelectValue placeholder="Select category..." />
                        </SelectTrigger>
                        <SelectContent>
                            {categories.map((cat) => (
                                <SelectItem key={cat.id} value={cat.id}>
                                    {cat.name}
                                </SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                </div>

                <div className="space-y-2">
                    <Label htmlFor="description">Description</Label>
                    <Textarea
                        id="description"
                        placeholder="Describe what happened..."
                        value={description}
                        onChange={(e) => setDescription(e.target.value)}
                        rows={4}
                    />
                </div>

                <div className="flex items-center gap-2">
                    <Switch id="urgent" checked={isUrgent} onCheckedChange={setIsUrgent} />
                    <Label htmlFor="urgent">Urgent — notify parent immediately</Label>
                </div>
            </div>

            <DialogFooter>
                <Button variant="outline" onClick={onSubmitSuccess}>
                    Cancel
                </Button>
                <Button
                    onClick={handleSubmit}
                    disabled={!categoryId || !description.trim() || createNote.isPending}
                >
                    {createNote.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                    Submit for Review
                </Button>
            </DialogFooter>
        </>
    );
}

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
