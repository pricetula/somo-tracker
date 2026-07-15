/**
 * ClassTeacherList — displays teacher assignments for a class or teacher.
 *
 * SCHOOL_ADMIN: can view by class or by teacher, and delete assignments.
 */
"use client";

import { useState } from "react";
import {
    useClassTeachersByClass,
    useClassTeachersByTeacher,
    useDeleteClassTeacher,
} from "../hooks/use-classteachers";
import { AssignTeacherDialog } from "./assign-teacher-dialog";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Plus, Trash2 } from "lucide-react";

type ViewMode = "by-class" | "by-teacher";

export function ClassTeacherList() {
    const [mode, setMode] = useState<ViewMode>("by-class");
    const [entityId, setEntityId] = useState("");
    const [showAssign, setShowAssign] = useState(false);

    const byClass = useClassTeachersByClass(mode === "by-class" ? entityId : "");
    const byTeacher = useClassTeachersByTeacher(mode === "by-teacher" ? entityId : "");
    const deleteMutation = useDeleteClassTeacher();

    const data = mode === "by-class" ? byClass.data : byTeacher.data;
    const isLoading = mode === "by-class" ? byClass.isLoading : byTeacher.isLoading;
    const items = data?.items ?? [];

    return (
        <div className="space-y-4">
            <div className="flex items-center justify-between gap-4">
                <div className="flex items-center gap-2">
                    <select
                        className="border-input bg-background rounded-md border px-3 py-1.5 text-sm"
                        value={mode}
                        onChange={(e) => setMode(e.target.value as ViewMode)}
                    >
                        <option value="by-class">By Class</option>
                        <option value="by-teacher">By Teacher</option>
                    </select>
                    <input
                        className="border-input bg-background w-64 rounded-md border px-3 py-1.5 text-sm"
                        placeholder={
                            mode === "by-class" ? "Enter Class ID…" : "Enter Teacher User ID…"
                        }
                        value={entityId}
                        onChange={(e) => setEntityId(e.target.value)}
                    />
                </div>
                {entityId && (
                    <Button variant="outline" size="sm" onClick={() => setShowAssign(true)}>
                        <Plus className="mr-1 size-4" />
                        Assign Teacher
                    </Button>
                )}
            </div>

            {isLoading ? (
                <p className="text-muted-foreground">Loading assignments…</p>
            ) : items.length === 0 ? (
                <p className="text-muted-foreground text-sm">
                    {entityId
                        ? "No assignments found."
                        : "Enter a class or teacher ID to view assignments."}
                </p>
            ) : (
                <div className="space-y-2">
                    {items.map((ct) => (
                        <div
                            key={ct.id}
                            className="bg-muted/30 flex items-center justify-between rounded-md p-3"
                        >
                            <div className="space-y-1">
                                <p className="text-foreground text-sm font-medium">
                                    {ct.teacher_name ?? ct.user_id}
                                </p>
                                <div className="text-muted-foreground flex items-center gap-2 text-xs">
                                    <Badge variant="secondary" className="text-xs">
                                        {ct.teacher_role.replace(/_/g, " ")}
                                    </Badge>
                                    {ct.learning_area && <span>{ct.learning_area}</span>}
                                </div>
                            </div>
                            <Button
                                variant="ghost"
                                size="icon"
                                className="size-6"
                                onClick={() => deleteMutation.mutate(ct.id)}
                            >
                                <Trash2 className="text-muted-foreground size-3" />
                            </Button>
                        </div>
                    ))}
                </div>
            )}

            <AssignTeacherDialog
                open={showAssign}
                onOpenChange={setShowAssign}
                prefillClassId={mode === "by-class" ? entityId : undefined}
            />
        </div>
    );
}
