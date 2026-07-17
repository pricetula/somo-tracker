/**
 * CreateScaleProfileForm — Creates a new grading scale profile with its
 * percentage-to-level ranges inline.
 *
 * The backend requires at least EE, ME, and AE ranges at creation time.
 * This eliminates the two-step workflow of creating a profile and then
 * separately adding ranges. After creation, ranges can be edited on the
 * profile detail page.
 */

"use client";

import { useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import { useMutation } from "@tanstack/react-query";
import { Loader2 } from "lucide-react";

import { createScaleProfile } from "@/lib/api/assessments";
import type { ScaleProfile } from "@/lib/api/assessments";
import { getErrorMessage, isApiError } from "@/lib/errors";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { PERFORMANCE_LEVELS, PERFORMANCE_LEVEL_LABELS } from "../types";

interface RangeInput {
    performance_level: string;
    min_percentage: string;
    max_percentage: string;
}

interface Props {
    onSuccess?: (profile: ScaleProfile) => void;
}

function defaultRanges(): RangeInput[] {
    return PERFORMANCE_LEVELS.map((level) => ({
        performance_level: level,
        min_percentage: "",
        max_percentage: "",
    }));
}

export function CreateScaleProfileForm({ onSuccess }: Props) {
    const router = useRouter();
    const [name, setName] = useState("");
    const [ranges, setRanges] = useState<RangeInput[]>(defaultRanges);
    const [error, setError] = useState<string | null>(null);

    const updateRange = useCallback(
        (level: string, field: "min_percentage" | "max_percentage", value: string) => {
            setRanges((prev) =>
                prev.map((r) => (r.performance_level === level ? { ...r, [field]: value } : r))
            );
            setError(null);
        },
        []
    );

    const createMutation = useMutation({
        mutationFn: () => {
            // ── Validate name ────────────────────────────────────────
            if (!name.trim()) throw new Error("Profile name is required.");

            // ── Parse and validate ranges ────────────────────────────
            const parsed = ranges.map((r) => ({
                ...r,
                minVal: parseFloat(r.min_percentage),
                maxVal: parseFloat(r.max_percentage),
            }));

            // Check required levels
            const required = ["EE", "ME", "AE"];
            const present = new Set(
                ranges
                    .filter((r) => r.min_percentage !== "" && r.max_percentage !== "")
                    .map((r) => r.performance_level)
            );
            for (const level of required) {
                if (!present.has(level)) {
                    throw new Error(
                        `${PERFORMANCE_LEVEL_LABELS[level]} (${level}) range is required.`
                    );
                }
            }

            // Validate each range
            for (const r of parsed) {
                if (r.min_percentage === "" || r.max_percentage === "") continue; // skipped levels are fine beyond required ones
                if (isNaN(r.minVal) || isNaN(r.maxVal)) {
                    throw new Error(
                        `Fill in both min and max for ${PERFORMANCE_LEVEL_LABELS[r.performance_level]}.`
                    );
                }
                if (r.minVal < 0 || r.maxVal > 100) {
                    throw new Error("Percentages must be between 0 and 100.");
                }
                if (r.maxVal <= r.minVal) {
                    throw new Error(
                        `${PERFORMANCE_LEVEL_LABELS[r.performance_level]}: max must be greater than min.`
                    );
                }
            }

            // Sort by min and check no overlaps
            const sorted = [...parsed]
                .filter((r) => r.min_percentage !== "")
                .sort((a, b) => a.minVal - b.minVal);
            for (let i = 1; i < sorted.length; i++) {
                if (sorted[i].minVal <= sorted[i - 1].maxVal) {
                    throw new Error(
                        `Ranges must not overlap. "${PERFORMANCE_LEVEL_LABELS[sorted[i].performance_level]}" starts at ${sorted[i].minVal}% but previous ends at ${sorted[i - 1].maxVal}%.`
                    );
                }
            }

            return createScaleProfile({
                name: name.trim(),
                ranges: parsed
                    .filter((r) => r.min_percentage !== "")
                    .map((r) => ({
                        performance_level: r.performance_level as "EE" | "ME" | "AE" | "BE",
                        min_percentage: r.minVal,
                        max_percentage: r.maxVal,
                    })),
            });
        },
        onSuccess: (result) => {
            router.push(`/assessments/grading-scales/${result.id}`);
            onSuccess?.(result as unknown as ScaleProfile);
        },
        onError: (err) => {
            if (isApiError(err) && err.status === 400 && err.errors) {
                setError(err.errors.name?.[0] ?? err.message);
            } else {
                setError(getErrorMessage(err));
            }
        },
    });

    const handleSubmit = useCallback(
        (e: React.FormEvent) => {
            e.preventDefault();
            setError(null);
            createMutation.mutate();
        },
        [createMutation]
    );

    return (
        <form onSubmit={handleSubmit} className="space-y-4">
            {error && (
                <Alert variant="destructive">
                    <AlertDescription>{error}</AlertDescription>
                </Alert>
            )}

            {/* Name */}
            <div className="space-y-1.5">
                <Label htmlFor="name">Profile Name</Label>
                <Input
                    id="name"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder='e.g. "Grade 4 Standard Conversion"'
                    disabled={createMutation.isPending}
                />
                <p className="text-muted-foreground text-xs">
                    Give this scale a descriptive name. Once created, the name cannot be changed.
                </p>
            </div>

            {/* Ranges */}
            <div className="space-y-1.5">
                <Label>Percentage Ranges</Label>
                <p className="text-muted-foreground text-xs">
                    Define the percentage boundaries for each CBC performance level. Ranges must be
                    contiguous with no gaps or overlaps.{" "}
                    <strong>EE, ME, and AE are required.</strong>
                </p>

                <div className="grid grid-cols-[1fr_80px_80px] items-end gap-3 pt-1">
                    <span className="text-muted-foreground text-xs font-medium">Level</span>
                    <span className="text-muted-foreground text-right text-xs font-medium">
                        Min %
                    </span>
                    <span className="text-muted-foreground text-right text-xs font-medium">
                        Max %
                    </span>

                    {ranges.map((r) => (
                        <div key={r.performance_level} className="contents">
                            <Label className="flex items-center gap-2 py-1">
                                <span className="font-semibold">{r.performance_level}</span>
                                <span className="text-muted-foreground font-normal">
                                    {PERFORMANCE_LEVEL_LABELS[r.performance_level]}
                                </span>
                            </Label>
                            <Input
                                type="number"
                                min={0}
                                max={100}
                                step={0.5}
                                value={r.min_percentage}
                                onChange={(e) =>
                                    updateRange(
                                        r.performance_level,
                                        "min_percentage",
                                        e.target.value
                                    )
                                }
                                className="h-8 text-right"
                                disabled={createMutation.isPending}
                                placeholder="0"
                            />
                            <Input
                                type="number"
                                min={0}
                                max={100}
                                step={0.5}
                                value={r.max_percentage}
                                onChange={(e) =>
                                    updateRange(
                                        r.performance_level,
                                        "max_percentage",
                                        e.target.value
                                    )
                                }
                                className="h-8 text-right"
                                disabled={createMutation.isPending}
                                placeholder="100"
                            />
                        </div>
                    ))}
                </div>
            </div>

            {/* Submit */}
            <div className="flex items-center justify-end gap-2 pt-2">
                <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => router.back()}
                    disabled={createMutation.isPending}
                >
                    Cancel
                </Button>
                <Button type="submit" size="sm" disabled={createMutation.isPending || !name.trim()}>
                    {createMutation.isPending ? (
                        <>
                            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                            Creating...
                        </>
                    ) : (
                        "Create Profile"
                    )}
                </Button>
            </div>
        </form>
    );
}
