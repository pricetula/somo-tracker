/**
 * SetScaleRangesForm — Allows an admin to define percentage-to-level ranges
 * for a grading scale profile.
 *
 * The four CBC levels (EE, ME, AE, BE) each get a min/max percentage input.
 * The form validates: no gaps/overlaps, min < max, ranges within 0-100.
 */

"use client";

import { useState, useCallback, useMemo } from "react";
import { Loader2 } from "lucide-react";

import { useScaleRanges, useBulkSetScaleRanges } from "../hooks/use-assessments";
import { PERFORMANCE_LEVELS, PERFORMANCE_LEVEL_LABELS } from "../types";
import { getErrorMessage } from "@/lib/errors";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Skeleton } from "@/components/ui/skeleton";

interface RangeInput {
    performance_level: string;
    min_percentage: string;
    max_percentage: string;
}

/** Build default ranges — either from existing API data or empty fields. */
function buildRanges(
    existingRanges:
        | { performance_level: string; min_percentage: number; max_percentage: number }[]
        | undefined
): RangeInput[] {
    return PERFORMANCE_LEVELS.map((level) => {
        const found = existingRanges?.find((r) => r.performance_level === level);
        return {
            performance_level: level,
            min_percentage: found ? String(found.min_percentage) : "",
            max_percentage: found ? String(found.max_percentage) : "",
        };
    });
}

interface Props {
    profileId: string;
}

export function SetScaleRangesForm({ profileId }: Props) {
    const { data: existingRanges, isLoading, isError } = useScaleRanges(profileId);
    const saveMutation = useBulkSetScaleRanges();

    // Derive initial ranges from API data — computed inline, no setState-in-effect
    const initialRanges = useMemo(() => buildRanges(existingRanges?.items), [existingRanges]);
    const [ranges, setRanges] = useState<RangeInput[]>(initialRanges);
    const [error, setError] = useState<string | null>(null);
    const [success, setSuccess] = useState(false);

    // Reset form when fresh API data arrives with actual values (user hasn't interacted yet)
    const [hasInteracted, setHasInteracted] = useState(false);
    if (
        !hasInteracted &&
        existingRanges?.items &&
        existingRanges.items.length > 0 &&
        ranges.every((r) => r.min_percentage === "" && r.max_percentage === "")
    ) {
        setRanges(initialRanges);
    }

    const updateRange = useCallback(
        (level: string, field: "min_percentage" | "max_percentage", value: string) => {
            setHasInteracted(true);
            setRanges((prev) =>
                prev.map((r) => (r.performance_level === level ? { ...r, [field]: value } : r))
            );
            setError(null);
            setSuccess(false);
        },
        []
    );

    const handleSubmit = useCallback(
        (e: React.FormEvent) => {
            e.preventDefault();
            setError(null);
            setSuccess(false);

            const parsed = ranges.map((r) => ({
                ...r,
                minVal: parseFloat(r.min_percentage),
                maxVal: parseFloat(r.max_percentage),
            }));

            for (const r of parsed) {
                if (isNaN(r.minVal) || isNaN(r.maxVal)) {
                    setError(
                        `Fill in both min and max for ${PERFORMANCE_LEVEL_LABELS[r.performance_level]}.`
                    );
                    return;
                }
                if (r.minVal < 0 || r.maxVal > 100) {
                    setError("Percentages must be between 0 and 100.");
                    return;
                }
                if (r.maxVal <= r.minVal) {
                    setError(
                        `${PERFORMANCE_LEVEL_LABELS[r.performance_level]}: max must be greater than min.`
                    );
                    return;
                }
            }

            const sorted = [...parsed].sort((a, b) => a.minVal - b.minVal);
            for (let i = 1; i < sorted.length; i++) {
                if (sorted[i].minVal <= sorted[i - 1].maxVal) {
                    setError(
                        `Ranges must not overlap. "${PERFORMANCE_LEVEL_LABELS[sorted[i].performance_level]}" starts at ${sorted[i].minVal}% but previous ends at ${sorted[i - 1].maxVal}%.`
                    );
                    return;
                }
            }

            saveMutation.mutate(
                {
                    profileId,
                    payload: {
                        ranges: parsed.map((r) => ({
                            performance_level: r.performance_level as "EE" | "ME" | "AE" | "BE",
                            min_percentage: r.minVal,
                            max_percentage: r.maxVal,
                        })),
                    },
                },
                {
                    onSuccess: () => setSuccess(true),
                    onError: (err) => setError(getErrorMessage(err)),
                }
            );
        },
        [ranges, profileId, saveMutation]
    );

    if (isLoading) {
        return (
            <div className="space-y-3">
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-10 w-full" />
            </div>
        );
    }

    if (isError) {
        return <p className="text-destructive text-sm">Failed to load existing ranges.</p>;
    }

    return (
        <form onSubmit={handleSubmit} className="space-y-4">
            {error && (
                <Alert variant="destructive">
                    <AlertDescription>{error}</AlertDescription>
                </Alert>
            )}
            {success && (
                <Alert>
                    <AlertDescription>Ranges saved successfully.</AlertDescription>
                </Alert>
            )}

            <p className="text-muted-foreground text-xs">
                Define the percentage boundaries for each CBC performance level. Ranges must be
                contiguous with no gaps or overlaps — e.g. BE: 0-39, AE: 40-59, ME: 60-79, EE:
                80-100.
            </p>

            <div className="grid grid-cols-[1fr_80px_80px] items-end gap-3">
                <span className="text-muted-foreground text-xs font-medium">Level</span>
                <span className="text-muted-foreground text-right text-xs font-medium">Min %</span>
                <span className="text-muted-foreground text-right text-xs font-medium">Max %</span>

                {ranges.map((r) => (
                    <div key={r.performance_level} className="contents">
                        <Label className="flex items-center gap-2 py-1 text-sm">
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
                                updateRange(r.performance_level, "min_percentage", e.target.value)
                            }
                            className="h-8 text-right"
                            disabled={saveMutation.isPending}
                        />
                        <Input
                            type="number"
                            min={0}
                            max={100}
                            step={0.5}
                            value={r.max_percentage}
                            onChange={(e) =>
                                updateRange(r.performance_level, "max_percentage", e.target.value)
                            }
                            className="h-8 text-right"
                            disabled={saveMutation.isPending}
                        />
                    </div>
                ))}
            </div>

            <div className="flex items-center justify-end gap-2 pt-2">
                <Button type="submit" size="sm" disabled={saveMutation.isPending}>
                    {saveMutation.isPending ? (
                        <>
                            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                            Saving...
                        </>
                    ) : (
                        "Save Ranges"
                    )}
                </Button>
            </div>
        </form>
    );
}
