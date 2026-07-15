/**
 * AssignTeacherDialog — modal to assign a teacher to a class.
 */
"use client";

import { useState } from "react";
import { useCreateClassTeacher } from "../hooks/use-classteachers";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { toast } from "sonner";

interface AssignTeacherDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    prefillClassId?: string;
}

export function AssignTeacherDialog({
    open,
    onOpenChange,
    prefillClassId,
}: AssignTeacherDialogProps) {
    const mutation = useCreateClassTeacher();
    const [classId, setClassId] = useState(prefillClassId ?? "");
    const [userId, setUserId] = useState("");
    const [teacherRole, setTeacherRole] = useState("SUBJECT_TEACHER");
    const [learningAreaId, setLearningAreaId] = useState("");

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!classId.trim() || !userId.trim()) {
            toast.error("Class ID and User ID are required.");
            return;
        }
        if (teacherRole === "SUBJECT_TEACHER" && !learningAreaId.trim()) {
            toast.error("Learning Area ID is required for subject teachers.");
            return;
        }
        try {
            await mutation.mutateAsync({
                class_id: classId.trim(),
                user_id: userId.trim(),
                teacher_role: teacherRole,
                learning_area_id: teacherRole === "SUBJECT_TEACHER" ? learningAreaId.trim() : null,
            });
            toast.success("Teacher assigned.");
            setUserId("");
            setLearningAreaId("");
            onOpenChange(false);
        } catch {
            toast.error("Failed to assign teacher.");
        }
    };

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="sm:max-w-md">
                <DialogHeader>
                    <DialogTitle>Assign Teacher to Class</DialogTitle>
                </DialogHeader>
                <form onSubmit={handleSubmit} className="space-y-4">
                    {!prefillClassId && (
                        <div className="space-y-1">
                            <Label htmlFor="class_id">Class ID</Label>
                            <Input
                                id="class_id"
                                value={classId}
                                onChange={(e) => setClassId(e.target.value)}
                                placeholder="Enter class UUID"
                            />
                        </div>
                    )}
                    <div className="space-y-1">
                        <Label htmlFor="user_id">Teacher User ID</Label>
                        <Input
                            id="user_id"
                            value={userId}
                            onChange={(e) => setUserId(e.target.value)}
                            placeholder="Enter teacher user UUID"
                        />
                    </div>
                    <div className="space-y-1">
                        <Label htmlFor="teacher_role">Role</Label>
                        <Select value={teacherRole} onValueChange={setTeacherRole}>
                            <SelectTrigger>
                                <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                                <SelectItem value="SUBJECT_TEACHER">Subject Teacher</SelectItem>
                                <SelectItem value="PRIMARY_CLASS_TEACHER">
                                    Primary Class Teacher
                                </SelectItem>
                                <SelectItem value="SUBSTITUTE_TEACHER">
                                    Substitute Teacher
                                </SelectItem>
                            </SelectContent>
                        </Select>
                    </div>
                    {teacherRole === "SUBJECT_TEACHER" && (
                        <div className="space-y-1">
                            <Label htmlFor="learning_area_id">Learning Area ID</Label>
                            <Input
                                id="learning_area_id"
                                value={learningAreaId}
                                onChange={(e) => setLearningAreaId(e.target.value)}
                                placeholder="Enter learning area UUID"
                            />
                        </div>
                    )}
                    <DialogFooter>
                        <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                            Cancel
                        </Button>
                        <Button type="submit" disabled={mutation.isPending}>
                            {mutation.isPending ? "Assigning…" : "Assign"}
                        </Button>
                    </DialogFooter>
                </form>
            </DialogContent>
        </Dialog>
    );
}
