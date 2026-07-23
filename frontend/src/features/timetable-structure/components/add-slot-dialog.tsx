/**
 * AddSlotDialog — dialog for assigning teacher, learning area, and room
 * to a time block slot. The class is pre-selected from the page's class
 * selector and passed in as a prop.
 */

"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Loader2, Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";
import { Combobox } from "@/components/ui/combobox";

import type { CreateSlotPayload } from "@/lib/api/timetable-structure";
import { useTeachers } from "@/features/teachers/hooks/use-teachers";
import { useLearningAreas } from "@/features/curriculum/hooks/use-curriculum";

// ─── Props ─────────────────────────────────────────────────────────────────

interface AddSlotDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    structureID: string;
    periodName: string;
    dayName: string;
    timeRange: string;
    academicYearID: string;
    isPending: boolean;
    /** Class ID from the page-level class selector. */
    classID: string;
    onSubmit: (payload: CreateSlotPayload) => void;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function AddSlotDialog({
    open,
    onOpenChange,
    structureID,
    periodName,
    dayName,
    timeRange,
    academicYearID,
    isPending,
    classID,
    onSubmit,
}: AddSlotDialogProps) {
    const router = useRouter();
    const { data: teachersData } = useTeachers({ limit: 200 });
    const { data: learningAreasData } = useLearningAreas({});

    const [teacherID, setTeacherID] = useState("");
    const [learningAreaID, setLearningAreaID] = useState("");
    const [room, setRoom] = useState("");

    const handleOpenChange = (open: boolean) => {
        if (open) {
            setTeacherID("");
            setLearningAreaID("");
            setRoom("");
        }
        onOpenChange(open);
    };

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        if (!classID) return;
        onSubmit({
            academic_year_id: academicYearID,
            structure_id: structureID,
            class_id: classID,
            teacher_id: teacherID || null,
            learning_area_id: learningAreaID || null,
            room_identifier: room || null,
        });
    };

    const teacherItems = (teachersData?.items ?? []).map((t) => ({
        value: t.id,
        label: t.full_name,
    }));

    const learningAreaItems = (learningAreasData?.items ?? []).map((la) => ({
        value: la.id,
        label: la.name,
    }));

    return (
        <Dialog open={open} onOpenChange={handleOpenChange}>
            <DialogContent className="sm:max-w-sm">
                <DialogHeader>
                    <DialogTitle>Assign Slot</DialogTitle>
                    <DialogDescription>
                        {dayName} &middot; {periodName} ({timeRange})
                    </DialogDescription>
                </DialogHeader>

                <form onSubmit={handleSubmit} className="space-y-4">
                    <div className="space-y-1.5">
                        <Label htmlFor="teacher">Teacher</Label>
                        <Combobox
                            items={teacherItems}
                            value={teacherID}
                            onValueChange={(v) => setTeacherID(v as string)}
                            placeholder="Select a teacher..."
                            emptyText="No teacher found."
                            className="w-full"
                        />
                    </div>

                    <div className="space-y-1.5">
                        <Label htmlFor="learning-area">Learning Area</Label>
                        <Combobox
                            items={learningAreaItems}
                            value={learningAreaID}
                            onValueChange={(v) => setLearningAreaID(v as string)}
                            placeholder="Select a learning area..."
                            emptyText="No learning area found."
                            className="w-full"
                            onCreateItem={() => router.push("/curriculum/new")}
                        />
                    </div>

                    <div className="space-y-1.5">
                        <Label htmlFor="room">Room</Label>
                        <Input
                            id="room"
                            type="text"
                            placeholder="e.g. Room 12, Lab A"
                            value={room}
                            onChange={(e) => setRoom(e.target.value)}
                        />
                    </div>

                    <DialogFooter className="pt-2">
                        <Button
                            type="button"
                            variant="outline"
                            onClick={() => onOpenChange(false)}
                            disabled={isPending}
                        >
                            Cancel
                        </Button>
                        <Button type="submit" disabled={isPending || !classID}>
                            {isPending ? (
                                <>
                                    <Loader2 className="mr-1.5 h-4 w-4 animate-spin" />
                                    Assigning…
                                </>
                            ) : (
                                <>
                                    <Plus className="mr-1.5 h-4 w-4" />
                                    Assign
                                </>
                            )}
                        </Button>
                    </DialogFooter>
                </form>
            </DialogContent>
        </Dialog>
    );
}
