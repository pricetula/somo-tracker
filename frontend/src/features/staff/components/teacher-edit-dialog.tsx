/**
 * TeacherEditDialog — edit a teacher's TSC number, KNEC panel assessor ID, and name.
 */

"use client";

import { useState, useEffect } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogFooter,
    DialogClose,
} from "@/components/ui/dialog";
import { getErrorMessage } from "@/lib/errors";
import { useTeacherDetail, useUpdateTeacher } from "../hooks/use-teachers";

interface TeacherEditDialogProps {
    userId: string | null;
    open: boolean;
    onOpenChange: (open: boolean) => void;
}

export function TeacherEditDialog({ userId, open, onOpenChange }: TeacherEditDialogProps) {
    const { data: teacher, isLoading, isError, error } = useTeacherDetail(userId ?? undefined);
    const updateMutation = useUpdateTeacher();

    const [fullName, setFullName] = useState("");
    const [tscNumber, setTscNumber] = useState("");
    const [knecAssessor, setKnecAssessor] = useState("");

    // Reset form when teacher data loads — syncs external API data into local form state.
    // This is a deliberate initialization effect, not cascading state updates.
    useEffect(() => {
        /* eslint-disable react-hooks/set-state-in-effect */
        if (teacher) {
            setFullName(teacher.full_name ?? "");
            setTscNumber(teacher.tsc_number ?? "");
            setKnecAssessor(teacher.knec_panel_assessor_id ?? "");
        }
        /* eslint-enable react-hooks/set-state-in-effect */
    }, [teacher]);

    function handleSave() {
        if (!userId) return;
        updateMutation.mutate(
            {
                userId,
                payload: {
                    full_name: fullName.trim() || undefined,
                    tsc_number: tscNumber.trim() || null,
                    knec_panel_assessor_id: knecAssessor.trim() || null,
                },
            },
            { onSuccess: () => onOpenChange(false) }
        );
    }

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent>
                <DialogHeader>
                    <DialogTitle>Edit Teacher</DialogTitle>
                </DialogHeader>

                {isLoading ? (
                    <div className="space-y-3 py-4">
                        <Skeleton className="h-5 w-32" />
                        <Skeleton className="h-9 w-full" />
                        <Skeleton className="h-5 w-32" />
                        <Skeleton className="h-9 w-full" />
                    </div>
                ) : isError ? (
                    <Alert variant="destructive">
                        <AlertDescription>{getErrorMessage(error)}</AlertDescription>
                    </Alert>
                ) : teacher ? (
                    <div className="space-y-4 py-2">
                        <div className="space-y-1.5">
                            <Label>Full Name</Label>
                            <Input
                                value={fullName}
                                onChange={(e) => setFullName(e.target.value)}
                                placeholder="Full name"
                            />
                        </div>
                        <div className="space-y-1.5">
                            <Label>TSC Number</Label>
                            <Input
                                value={tscNumber}
                                onChange={(e) => setTscNumber(e.target.value)}
                                placeholder="e.g. TSC123456"
                            />
                        </div>
                        <div className="space-y-1.5">
                            <Label>KNEC Panel Assessor ID</Label>
                            <Input
                                value={knecAssessor}
                                onChange={(e) => setKnecAssessor(e.target.value)}
                                placeholder="e.g. KNEC-12345"
                            />
                        </div>
                        {updateMutation.error && (
                            <p className="text-destructive">
                                {getErrorMessage(updateMutation.error)}
                            </p>
                        )}
                    </div>
                ) : (
                    <p className="text-muted-foreground py-4">Teacher not found.</p>
                )}

                <DialogFooter>
                    <DialogClose asChild>
                        <Button variant="outline">Cancel</Button>
                    </DialogClose>
                    <Button onClick={handleSave} disabled={updateMutation.isPending || !teacher}>
                        {updateMutation.isPending ? "Saving…" : "Save"}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}
