/**
 * CreateIncidentDialog — modal to log a new medical incident.
 */
"use client";

import { useState } from "react";
import { useCreateMedicalIncident } from "../hooks/use-health";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { StudentCombobox } from "@/features/students/components/student-combobox";
import { toast } from "sonner";

interface CreateIncidentDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    studentId?: string;
}

export function CreateIncidentDialog({ open, onOpenChange, studentId }: CreateIncidentDialogProps) {
    const mutation = useCreateMedicalIncident();
    const [studentIdInput, setStudentIdInput] = useState(studentId ?? "");
    const [symptoms, setSymptoms] = useState("");
    const [actionTaken, setActionTaken] = useState("");

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!studentIdInput.trim() || !symptoms.trim() || !actionTaken.trim()) {
            toast.error("All fields are required.");
            return;
        }
        try {
            await mutation.mutateAsync({
                student_id: studentIdInput.trim(),
                symptoms: symptoms.trim(),
                action_taken: actionTaken.trim(),
            });
            toast.success("Incident logged.");
            setSymptoms("");
            setActionTaken("");
            onOpenChange(false);
        } catch {
            toast.error("Failed to log incident.");
        }
    };

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="sm:max-w-lg">
                <DialogHeader>
                    <DialogTitle>Log Medical Incident</DialogTitle>
                </DialogHeader>
                <form onSubmit={handleSubmit} className="space-y-4">
                    {!studentId && (
                        <div className="space-y-1">
                            <Label htmlFor="student_id">Student</Label>
                            <StudentCombobox
                                value={studentIdInput}
                                onChange={(v) => setStudentIdInput(v as string)}
                                placeholder="Select a student..."
                            />
                        </div>
                    )}
                    <div className="space-y-1">
                        <Label htmlFor="symptoms">Symptoms / Description</Label>
                        <Textarea
                            id="symptoms"
                            value={symptoms}
                            onChange={(e) => setSymptoms(e.target.value)}
                            placeholder="Describe the symptoms or incident…"
                            rows={3}
                        />
                    </div>
                    <div className="space-y-1">
                        <Label htmlFor="action_taken">Action Taken</Label>
                        <Textarea
                            id="action_taken"
                            value={actionTaken}
                            onChange={(e) => setActionTaken(e.target.value)}
                            placeholder="Describe the action taken…"
                            rows={3}
                        />
                    </div>
                    <DialogFooter>
                        <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                            Cancel
                        </Button>
                        <Button type="submit" disabled={mutation.isPending}>
                            {mutation.isPending ? "Saving…" : "Log Incident"}
                        </Button>
                    </DialogFooter>
                </form>
            </DialogContent>
        </Dialog>
    );
}
