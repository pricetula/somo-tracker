/**
 * Link Student Dialog — modal to search and link a student to a parent.
 *
 * Features:
 * - Search students by name
 * - Select relationship type (optional)
 * - Toggle primary guardian status
 */

"use client";

import * as React from "react";

import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
} from "@/components/ui/dialog";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";
import { Loader2, Search } from "lucide-react";

import { useLinkStudent } from "../hooks/use-parents";
import { useStudents } from "@/features/students";
import { getErrorMessage } from "@/lib/errors";

// ─── Props ─────────────────────────────────────────────────────────────────

interface LinkStudentDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    parentId: string;
}

const RELATIONSHIP_OPTIONS = [
    "Mother",
    "Father",
    "Guardian",
    "Grandparent",
    "Sibling",
    "Aunt",
    "Uncle",
    "Other",
] as const;

// ─── Component ─────────────────────────────────────────────────────────────

export function LinkStudentDialog({ open, onOpenChange, parentId }: LinkStudentDialogProps) {
    const [search, setSearch] = React.useState("");
    const [selectedStudentId, setSelectedStudentId] = React.useState("");
    const [relationship, setRelationship] = React.useState("");
    const [isPrimary, setIsPrimary] = React.useState(false);
    const [error, setError] = React.useState<string | null>(null);

    const linkStudent = useLinkStudent();

    const { data: studentsData, isLoading: studentsLoading } = useStudents(
        { search: search || undefined, limit: 50 },
        { enabled: open }
    );

    const students = studentsData?.students ?? [];

    // Reset state via onOpenChange instead of an effect

    const handleLink = async () => {
        if (!selectedStudentId) {
            setError("Please select a student");
            return;
        }

        setError(null);

        try {
            await linkStudent.mutateAsync({
                parentId,
                data: {
                    student_id: selectedStudentId,
                    relationship: relationship || null,
                    is_primary: isPrimary || undefined,
                },
            });
            onOpenChange(false);
        } catch (err) {
            setError(getErrorMessage(err));
        }
    };

    const handleOpenChange = (next: boolean) => {
        if (next) {
            setSearch("");
            setSelectedStudentId("");
            setRelationship("");
            setIsPrimary(false);
            setError(null);
        }
        onOpenChange(next);
    };

    const selectedStudent = students.find((s) => s.id === selectedStudentId);

    return (
        <Dialog open={open} onOpenChange={handleOpenChange}>
            <DialogContent className="max-w-md">
                <DialogHeader>
                    <DialogTitle>Link Student</DialogTitle>
                    <DialogDescription>
                        Search and select a student to link to this parent.
                    </DialogDescription>
                </DialogHeader>

                <div className="space-y-4">
                    {error && (
                        <div className="text-destructive bg-destructive/10 rounded-md px-3 py-2 text-sm">
                            {error}
                        </div>
                    )}

                    {/* Student search */}
                    <div className="space-y-1.5">
                        <Label>Search Student</Label>
                        <div className="relative">
                            <Search className="text-muted-foreground absolute top-2.5 left-2.5 size-4" />
                            <Input
                                placeholder="Type student name…"
                                value={search}
                                onChange={(e) => {
                                    setSearch(e.target.value);
                                    setSelectedStudentId("");
                                }}
                                className="pl-8"
                            />
                        </div>
                    </div>

                    {/* Student results */}
                    <div className="min-h-[120px]">
                        {studentsLoading ? (
                            <div className="flex items-center justify-center py-8">
                                <Loader2 className="text-muted-foreground size-5 animate-spin" />
                            </div>
                        ) : students.length === 0 ? (
                            <p className="text-muted-foreground py-4 text-center text-sm">
                                {search
                                    ? "No students match your search"
                                    : "Type to search for students"}
                            </p>
                        ) : (
                            <div className="border-border/40 max-h-48 space-y-1 overflow-auto rounded-md border">
                                {students.map((s) => {
                                    const isSelected = s.id === selectedStudentId;
                                    return (
                                        <button
                                            key={s.id}
                                            type="button"
                                            className={`hover:bg-muted/50 flex w-full items-center gap-2 px-3 py-2 text-left text-sm transition-colors ${
                                                isSelected ? "bg-muted font-medium" : ""
                                            }`}
                                            onClick={() => setSelectedStudentId(s.id)}
                                        >
                                            <span className="flex-1 truncate">{s.full_name}</span>
                                            {s.class_name && (
                                                <span className="text-muted-foreground text-xs">
                                                    {s.class_name}
                                                </span>
                                            )}
                                            {isSelected && (
                                                <Badge
                                                    variant="secondary"
                                                    className="bg-sky-100 text-[10px] text-sky-700 dark:bg-sky-900/30 dark:text-sky-400"
                                                >
                                                    Selected
                                                </Badge>
                                            )}
                                        </button>
                                    );
                                })}
                            </div>
                        )}
                    </div>

                    {/* Selected student summary */}
                    {selectedStudent && (
                        <div className="bg-muted/30 rounded-md px-3 py-2 text-sm">
                            <span className="font-medium">Selected: </span>
                            {selectedStudent.full_name}
                            {selectedStudent.class_name && (
                                <span className="text-muted-foreground">
                                    {" "}
                                    — {selectedStudent.class_name}
                                </span>
                            )}
                        </div>
                    )}

                    {/* Relationship */}
                    <div className="space-y-1.5">
                        <Label htmlFor="relationship">Relationship (optional)</Label>
                        <Select value={relationship} onValueChange={setRelationship}>
                            <SelectTrigger id="relationship">
                                <SelectValue placeholder="Select relationship" />
                            </SelectTrigger>
                            <SelectContent>
                                {RELATIONSHIP_OPTIONS.map((r) => (
                                    <SelectItem key={r} value={r}>
                                        {r}
                                    </SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                    </div>

                    {/* Is Primary */}
                    <div className="flex items-center gap-3">
                        <Switch
                            id="is_primary"
                            checked={isPrimary}
                            onCheckedChange={setIsPrimary}
                        />
                        <Label htmlFor="is_primary" className="text-sm">
                            Primary guardian
                        </Label>
                    </div>

                    {/* Actions */}
                    <div className="flex items-center justify-end gap-3 pt-2">
                        <Button
                            variant="ghost"
                            onClick={() => onOpenChange(false)}
                            disabled={linkStudent.isPending}
                        >
                            Cancel
                        </Button>
                        <Button
                            onClick={handleLink}
                            disabled={!selectedStudentId || linkStudent.isPending}
                        >
                            {linkStudent.isPending ? (
                                <>
                                    <Loader2 className="mr-1.5 size-4 animate-spin" />
                                    Linking…
                                </>
                            ) : (
                                "Link Student"
                            )}
                        </Button>
                    </div>
                </div>
            </DialogContent>
        </Dialog>
    );
}
