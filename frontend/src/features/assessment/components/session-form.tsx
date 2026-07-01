/**
 * Session Form — create a new assessment session.
 *
 * Step 1: Select blueprint (from teacher's school blueprints)
 * Step 2: Select class, choose date
 * Submit → creates session, navigates to score page
 */

"use client";

import * as React from "react";
import { useRouter } from "next/navigation";

import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Calendar } from "@/components/ui/calendar";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { CalendarIcon, Loader2 } from "lucide-react";

import { useBlueprints, useCreateSession } from "../hooks/use-assessment";
import { getErrorMessage } from "@/lib/errors";

// ─── Class type (from generated.ts) ───────────────────────────────────────

interface ClassOption {
    id: string;
    display_label: string;
    grade_level: string;
    stream_name: string;
    student_count?: number;
}

interface SessionFormProps {
    classes: ClassOption[];
    classesLoading: boolean;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function SessionForm({ classes, classesLoading }: SessionFormProps) {
    const router = useRouter();
    const createSession = useCreateSession();

    const { data: blueprintsData, isLoading: blueprintsLoading } = useBlueprints();

    const [blueprintId, setBlueprintId] = React.useState("");
    const [classId, setClassId] = React.useState("");
    const [date, setDate] = React.useState<Date | undefined>(undefined);
    const [error, setError] = React.useState<string | null>(null);

    const blueprints = blueprintsData?.data ?? [];

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setError(null);

        if (!blueprintId) {
            setError("Please select a blueprint");
            return;
        }
        if (!classId) {
            setError("Please select a class");
            return;
        }
        if (!date) {
            setError("Please select a date");
            return;
        }

        const dateISO = date.toISOString().split("T")[0];

        try {
            const result = await createSession.mutateAsync({
                blueprint_id: blueprintId,
                class_id: classId,
                date_administered: dateISO,
            });

            router.push(`/assessment/sessions/${result.id}/score`);
        } catch (err) {
            setError(getErrorMessage(err));
        }
    };

    // Check for empty class warnings
    const selectedClass = classes.find((c) => c.id === classId);
    const hasNoStudentsWarning = selectedClass && selectedClass.student_count === 0;
    const isSubmitting = createSession.isPending;

    const dateString = date
        ? date.toLocaleDateString("en-US", {
              month: "long",
              day: "numeric",
              year: "numeric",
          })
        : "";

    return (
        <form onSubmit={handleSubmit} className="space-y-5">
            {error && (
                <div className="text-destructive bg-destructive/10 rounded-md px-3 py-2 text-sm">
                    {error}
                </div>
            )}

            {/* Blueprint selection */}
            <div className="space-y-1.5">
                <Label htmlFor="blueprint_id">Assessment Blueprint</Label>
                {blueprintsLoading ? (
                    <Skeleton className="h-10 w-full" />
                ) : blueprints.length === 0 ? (
                    <p className="text-muted-foreground text-sm">
                        No blueprints available.{" "}
                        <Button
                            variant="link"
                            className="h-auto p-0 text-xs"
                            onClick={() => router.push("/assessment/blueprints/new")}
                        >
                            Create one first.
                        </Button>
                    </p>
                ) : (
                    <Select
                        value={blueprintId}
                        onValueChange={setBlueprintId}
                        disabled={isSubmitting}
                    >
                        <SelectTrigger id="blueprint_id">
                            <SelectValue placeholder="Select a blueprint" />
                        </SelectTrigger>
                        <SelectContent>
                            {blueprints.map((bp) => (
                                <SelectItem key={bp.id} value={bp.id}>
                                    {bp.title} — {bp.grade_level}, Term {bp.term}
                                </SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                )}
            </div>

            {/* Class selection */}
            <div className="space-y-1.5">
                <Label htmlFor="class_id">Class</Label>
                {classesLoading ? (
                    <Skeleton className="h-10 w-full" />
                ) : classes.length === 0 ? (
                    <p className="text-muted-foreground text-sm">
                        No classes available. Classes must be configured first.
                    </p>
                ) : (
                    <Select value={classId} onValueChange={setClassId} disabled={isSubmitting}>
                        <SelectTrigger id="class_id">
                            <SelectValue placeholder="Select a class" />
                        </SelectTrigger>
                        <SelectContent>
                            {classes.map((c) => (
                                <SelectItem key={c.id} value={c.id}>
                                    {c.display_label}
                                    {c.student_count !== undefined
                                        ? ` (${c.student_count} students)`
                                        : ""}
                                </SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                )}
                {hasNoStudentsWarning && (
                    <p className="text-xs text-amber-600 dark:text-amber-400">
                        This class has no enrolled students. Add students before scoring.
                    </p>
                )}
            </div>

            {/* Date picker */}
            <div className="space-y-1.5">
                <Label>Date Administered</Label>
                <Popover>
                    <PopoverTrigger asChild>
                        <Button
                            variant="outline"
                            className="w-full justify-start text-left font-normal"
                            disabled={isSubmitting}
                        >
                            <CalendarIcon className="mr-2 size-4" />
                            {date ? (
                                dateString
                            ) : (
                                <span className="text-muted-foreground">Pick a date</span>
                            )}
                        </Button>
                    </PopoverTrigger>
                    <PopoverContent className="w-auto p-0" align="start">
                        <Calendar mode="single" selected={date} onSelect={setDate} />
                    </PopoverContent>
                </Popover>
            </div>

            {/* Submit */}
            <div className="flex items-center gap-3 pt-2">
                <Button type="submit" disabled={isSubmitting || !blueprintId || !classId || !date}>
                    {isSubmitting ? (
                        <>
                            <Loader2 className="mr-1.5 size-4 animate-spin" />
                            Creating…
                        </>
                    ) : (
                        "Create Session & Start Scoring"
                    )}
                </Button>
                <Button
                    type="button"
                    variant="ghost"
                    onClick={() => router.back()}
                    disabled={isSubmitting}
                >
                    Cancel
                </Button>
            </div>
        </form>
    );
}
