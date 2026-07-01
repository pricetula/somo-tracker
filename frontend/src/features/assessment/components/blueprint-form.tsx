/**
 * Blueprint Form — create a new assessment blueprint.
 *
 * Fields: Title, Type, Grade Level, Academic Year, Term.
 * On success navigates to the blueprint detail page for indicator linking.
 */

"use client";

import * as React from "react";
import { useRouter } from "next/navigation";

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

import { useCreateBlueprint } from "../hooks/use-assessment";
import { ASSESSMENT_TYPES, GRADE_LEVELS } from "../types";
import { getErrorMessage } from "@/lib/errors";

// ─── Props ─────────────────────────────────────────────────────────────────

interface BlueprintFormProps {
    onSuccess?: (id: string) => void;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function BlueprintForm({ onSuccess }: BlueprintFormProps) {
    const router = useRouter();
    const createBlueprint = useCreateBlueprint();

    const [title, setTitle] = React.useState("");
    const [type, setType] = React.useState("");
    const [gradeLevel, setGradeLevel] = React.useState("");
    const [academicYear, setAcademicYear] = React.useState(new Date().getFullYear());
    const [term, setTerm] = React.useState(1);
    const [error, setError] = React.useState<string | null>(null);

    const currentYear = new Date().getFullYear();

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setError(null);

        if (!title.trim()) {
            setError("Title is required");
            return;
        }
        if (!type) {
            setError("Assessment type is required");
            return;
        }
        if (!gradeLevel) {
            setError("Grade level is required");
            return;
        }

        try {
            const result = await createBlueprint.mutateAsync({
                title: title.trim(),
                type,
                grade_level: gradeLevel,
                academic_year: academicYear,
                term,
            });

            if (onSuccess) {
                onSuccess(result.id);
            } else {
                router.push(`/assessment/blueprints/${result.id}`);
            }
        } catch (err) {
            setError(getErrorMessage(err));
        }
    };

    const isSubmitting = createBlueprint.isPending;

    return (
        <form onSubmit={handleSubmit} className="space-y-5">
            {error && (
                <div className="text-destructive bg-destructive/10 rounded-md px-3 py-2 text-sm">
                    {error}
                </div>
            )}

            {/* Title */}
            <div className="space-y-1.5">
                <Label htmlFor="title">Title</Label>
                <Input
                    id="title"
                    value={title}
                    onChange={(e) => setTitle(e.target.value)}
                    placeholder="e.g. End of Term 1 Mathematics Assessment"
                    disabled={isSubmitting}
                />
            </div>

            {/* Type */}
            <div className="space-y-1.5">
                <Label htmlFor="type">Assessment Type</Label>
                <Select value={type} onValueChange={setType} disabled={isSubmitting}>
                    <SelectTrigger id="type">
                        <SelectValue placeholder="Select type" />
                    </SelectTrigger>
                    <SelectContent>
                        {ASSESSMENT_TYPES.map((t) => (
                            <SelectItem key={t} value={t}>
                                {t.replace(/_/g, " ")}
                            </SelectItem>
                        ))}
                    </SelectContent>
                </Select>
            </div>

            {/* Grade Level */}
            <div className="space-y-1.5">
                <Label htmlFor="grade_level">Grade Level</Label>
                <Select value={gradeLevel} onValueChange={setGradeLevel} disabled={isSubmitting}>
                    <SelectTrigger id="grade_level">
                        <SelectValue placeholder="Select grade" />
                    </SelectTrigger>
                    <SelectContent>
                        {GRADE_LEVELS.map((g) => (
                            <SelectItem key={g} value={g}>
                                {g}
                            </SelectItem>
                        ))}
                    </SelectContent>
                </Select>
            </div>

            {/* Academic Year + Term side by side */}
            <div className="grid grid-cols-2 gap-4">
                <div className="space-y-1.5">
                    <Label htmlFor="academic_year">Academic Year</Label>
                    <Input
                        id="academic_year"
                        type="number"
                        min={2017}
                        max={currentYear + 1}
                        value={academicYear}
                        onChange={(e) => setAcademicYear(Number(e.target.value))}
                        disabled={isSubmitting}
                    />
                </div>
                <div className="space-y-1.5">
                    <Label htmlFor="term">Term</Label>
                    <Select
                        value={String(term)}
                        onValueChange={(v) => setTerm(Number(v))}
                        disabled={isSubmitting}
                    >
                        <SelectTrigger id="term">
                            <SelectValue placeholder="Select term" />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem value="1">Term 1</SelectItem>
                            <SelectItem value="2">Term 2</SelectItem>
                            <SelectItem value="3">Term 3</SelectItem>
                        </SelectContent>
                    </Select>
                </div>
            </div>

            {/* Submit */}
            <div className="flex items-center gap-3 pt-2">
                <Button type="submit" disabled={isSubmitting}>
                    {isSubmitting ? "Creating…" : "Create Blueprint"}
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
