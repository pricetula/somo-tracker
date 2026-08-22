"use client";

import { useState, useEffect, useCallback } from "react";
import { X, Check, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
    DialogFooter,
} from "@/components/ui/dialog";
import { Alert, AlertDescription } from "@/components/ui/alert";
import type { CreateSlotPayload, UpdateSlotPayload, EnrichedSlot, SlotFormState } from "../types";

interface SlotAssignmentFormProps {
    open: boolean;
    onClose: () => void;
    onSubmit: (payload: CreateSlotPayload | UpdateSlotPayload) => Promise<void>;
    structureId: string;
    initialData?: {
        classId: string;
        learningAreaId: string;
        teacherId: string;
        roomIdentifier?: string;
    };
    editingSlot?: EnrichedSlot;
    classes: Array<{ id: string; name: string }>;
    learningAreas: Array<{ id: string; name: string }>;
    teachers: Array<{ id: string; name: string }>;
    rooms: Array<{ id: string; identifier: string }>;
    isLoading?: boolean;
}

export function SlotAssignmentForm({
    open,
    onClose,
    onSubmit,
    structureId,
    initialData,
    editingSlot,
    classes,
    learningAreas,
    teachers,
    rooms,
    isLoading = false,
}: SlotAssignmentFormProps) {
    // Use a key-like approach: derive initial form data from props
    // This avoids setState in effect by computing initial values directly
    const getInitialFormData = useCallback((): SlotFormState => {
        if (editingSlot) {
            return {
                structureId,
                classId: editingSlot.class_id ?? "",
                learningAreaId: (editingSlot.learning_area_id ?? "") as string,
                teacherId: (editingSlot.teacher_id ?? "") as string,
                roomIdentifier: (editingSlot.room_identifier ?? "") as string,
            };
        }
        if (initialData) {
            return {
                structureId,
                classId: initialData.classId ?? "",
                learningAreaId: initialData.learningAreaId ?? "",
                teacherId: initialData.teacherId ?? "",
                roomIdentifier: initialData.roomIdentifier ?? "",
            };
        }
        return {
            structureId,
            classId: "",
            learningAreaId: "",
            teacherId: "",
            roomIdentifier: "",
        };
    }, [editingSlot, initialData, structureId]);

    const [formData, setFormData] = useState<SlotFormState>(getInitialFormData);
    const [errors, setErrors] = useState<Record<string, string>>({});
    const [submitError, setSubmitError] = useState<string | null>(null);

    // Reset form when key props change (dialog reopens with different data)
    // Using functional updates to avoid lint warning about setState in effect
    useEffect(() => {
        if (open) {
            // eslint-disable-next-line react-hooks/set-state-in-effect
            setFormData(getInitialFormData);
            setErrors({});
            setSubmitError(null);
        }
    }, [open, editingSlot, initialData, structureId, getInitialFormData]);

    const validate = () => {
        const newErrors: Record<string, string> = {};
        if (!formData.classId) newErrors.classId = "Class is required";
        if (!formData.learningAreaId) newErrors.learningAreaId = "Learning area is required";
        if (!formData.teacherId) newErrors.teacherId = "Teacher is required";
        setErrors(newErrors);
        return Object.keys(newErrors).length === 0;
    };

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!validate()) return;

        setSubmitError(null);
        try {
            if (editingSlot) {
                const updatePayload: UpdateSlotPayload = {
                    learningAreaId: formData.learningAreaId || undefined,
                    teacherId: formData.teacherId || undefined,
                    roomIdentifier: formData.roomIdentifier || undefined,
                };
                await onSubmit(updatePayload);
            } else {
                const createPayload: CreateSlotPayload = {
                    structureId: formData.structureId,
                    classId: formData.classId,
                    learningAreaId: formData.learningAreaId,
                    teacherId: formData.teacherId,
                    roomIdentifier: formData.roomIdentifier || undefined,
                };
                await onSubmit(createPayload);
            }
            onClose();
        } catch (error: unknown) {
            const apiError = error as {
                code?: string;
                message?: string;
                errors?: Record<string, string[]>;
            };
            if (apiError.errors) {
                const fieldErrors: Record<string, string> = {};
                for (const [field, messages] of Object.entries(apiError.errors)) {
                    fieldErrors[field] = messages[0];
                }
                setErrors(fieldErrors);
            }
            setSubmitError(apiError.message ?? "Failed to save slot assignment");
        }
    };

    return (
        <Dialog open={open} onOpenChange={onClose}>
            <DialogContent className="sm:max-w-[500px]">
                <DialogHeader>
                    <DialogTitle>
                        {editingSlot ? "Edit Lesson Assignment" : "Assign Lesson"}
                    </DialogTitle>
                    <DialogDescription>
                        Select the class, subject, teacher, and room for this period.
                    </DialogDescription>
                </DialogHeader>
                <form id="assignment-form" onSubmit={handleSubmit} className="space-y-4">
                    {submitError && (
                        <Alert variant="destructive">
                            <AlertDescription>{submitError}</AlertDescription>
                        </Alert>
                    )}

                    <div className="space-y-2">
                        <Label htmlFor="class">Class *</Label>
                        <Select
                            value={formData.classId as string}
                            onValueChange={(v) =>
                                setFormData({ ...formData, classId: v as string })
                            }
                            disabled={!!editingSlot}
                        >
                            <SelectTrigger id="class">
                                <SelectValue placeholder="Select class" />
                            </SelectTrigger>
                            <SelectContent>
                                {classes.map((c) => (
                                    <SelectItem key={c.id} value={c.id}>
                                        {c.name}
                                    </SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                        {errors.classId && (
                            <p className="text-destructive text-sm">{errors.classId}</p>
                        )}
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor="learningArea">Learning Area *</Label>
                        <Select
                            value={formData.learningAreaId as string}
                            onValueChange={(v) =>
                                setFormData({ ...formData, learningAreaId: v as string })
                            }
                        >
                            <SelectTrigger id="learningArea">
                                <SelectValue placeholder="Select learning area" />
                            </SelectTrigger>
                            <SelectContent>
                                {learningAreas.map((la) => (
                                    <SelectItem key={la.id} value={la.id}>
                                        {la.name}
                                    </SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                        {errors.learningAreaId && (
                            <p className="text-destructive text-sm">{errors.learningAreaId}</p>
                        )}
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor="teacher">Teacher *</Label>
                        <Select
                            value={formData.teacherId as string}
                            onValueChange={(v) =>
                                setFormData({ ...formData, teacherId: v as string })
                            }
                        >
                            <SelectTrigger id="teacher">
                                <SelectValue placeholder="Select teacher" />
                            </SelectTrigger>
                            <SelectContent>
                                {teachers.map((t) => (
                                    <SelectItem key={t.id} value={t.id}>
                                        {t.name}
                                    </SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                        {errors.teacherId && (
                            <p className="text-destructive text-sm">{errors.teacherId}</p>
                        )}
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor="room">Room (Optional)</Label>
                        <Select
                            value={formData.roomIdentifier as string}
                            onValueChange={(v) =>
                                setFormData({ ...formData, roomIdentifier: v as string })
                            }
                        >
                            <SelectTrigger id="room">
                                <SelectValue placeholder="Select room (optional)" />
                            </SelectTrigger>
                            <SelectContent>
                                <SelectItem value="">None</SelectItem>
                                {rooms.map((r) => (
                                    <SelectItem key={r.id} value={r.identifier}>
                                        {r.identifier}
                                    </SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                    </div>
                </form>
                <DialogFooter>
                    <Button type="button" variant="outline" onClick={onClose}>
                        <X className="mr-2 h-4 w-4" />
                        Cancel
                    </Button>
                    <Button type="submit" form="assignment-form" disabled={isLoading}>
                        {isLoading ? (
                            <>
                                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                                Saving...
                            </>
                        ) : (
                            <>
                                <Check className="mr-2 h-4 w-4" />
                                {editingSlot ? "Update" : "Assign"}
                            </>
                        )}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}
