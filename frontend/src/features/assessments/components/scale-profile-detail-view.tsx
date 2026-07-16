/**
 * ScaleProfileDetailView — Shows a grading scale profile with its ranges.
 * Used by the full page and the intercepted side sheet (if needed).
 *
 * Admin can view the defined ranges and toggle active status.
 */

"use client";

import { useQuery } from "@tanstack/react-query";
import { ToggleLeft, ToggleRight } from "lucide-react";

import { getScaleProfile } from "@/lib/api/assessments";
import type { ScaleProfileWithRanges } from "@/lib/api/assessments";
import { useToggleScaleProfile } from "../hooks/use-assessments";
import { PERFORMANCE_LEVEL_LABELS } from "../types";
import { getErrorMessage } from "@/lib/errors";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { StaticTable } from "@/components/shared/static-table";
import { SetScaleRangesForm } from "./set-scale-ranges-form";
import { toast } from "sonner";

interface Props {
    profileId: string;
}

function useProfileWithRanges(id: string) {
    return useQuery({
        queryKey: ["scale-profiles", id, "with-ranges"],
        queryFn: () => getScaleProfile(id, true) as Promise<ScaleProfileWithRanges>,
        enabled: !!id,
        staleTime: 30_000,
    });
}

export function ScaleProfileDetailView({ profileId }: Props) {
    const { data: profile, isLoading, isError } = useProfileWithRanges(profileId);
    const toggleMutation = useToggleScaleProfile();

    if (isLoading) {
        return (
            <div className="space-y-4">
                <Skeleton className="h-8 w-64" />
                <Skeleton className="h-40 w-full" />
            </div>
        );
    }

    if (isError || !profile) {
        return (
            <p className="text-destructive py-8 text-center">
                {isError ? "Failed to load profile." : "Profile not found."}
            </p>
        );
    }

    return (
        <article className="space-y-6">
            {/* Header */}
            <div className="flex items-start justify-between gap-4">
                <div>
                    <h1 className="text-lg font-semibold">{profile.name}</h1>
                    <p className="text-muted-foreground mt-0.5 text-xs">
                        Created {new Date(profile.created_at).toLocaleDateString()}
                    </p>
                </div>
                <div className="flex items-center gap-2">
                    <Badge
                        variant="secondary"
                        className={
                            profile.is_active
                                ? "bg-emerald-100 text-emerald-700"
                                : "bg-muted text-muted-foreground"
                        }
                    >
                        {profile.is_active ? "Active" : "Inactive"}
                    </Badge>
                    <Button
                        variant="outline"
                        size="sm"
                        onClick={() =>
                            toggleMutation.mutate(
                                { id: profileId, isActive: !profile.is_active },
                                {
                                    onError: (err) => toast.error(getErrorMessage(err)),
                                }
                            )
                        }
                        disabled={toggleMutation.isPending}
                    >
                        {profile.is_active ? (
                            <ToggleLeft className="mr-1.5 h-4 w-4" />
                        ) : (
                            <ToggleRight className="mr-1.5 h-4 w-4" />
                        )}
                        {profile.is_active ? "Deactivate" : "Activate"}
                    </Button>
                </div>
            </div>

            <p className="text-muted-foreground text-xs">
                Profiles are immutable once created. To change a scale, deactivate this profile and
                create a new one. The percentage ranges below define how numeric scores map to CBC
                rubric levels. Ranges must be contiguous with no gaps or overlaps.
            </p>

            {/* Existing ranges (read-only table) */}
            {profile.ranges && profile.ranges.length > 0 && (
                <div className="space-y-2">
                    <h2 className="font-medium">Current Ranges</h2>
                    <StaticTable
                        columns={[
                            { id: "level", header: "Level", cell: (r) => r.performance_level },
                            {
                                id: "label",
                                header: "Description",
                                cell: (r) =>
                                    PERFORMANCE_LEVEL_LABELS[r.performance_level] ??
                                    r.performance_level,
                            },
                            {
                                id: "min",
                                header: "Min %",
                                align: "right",
                                cell: (r) => `${r.min_percentage}%`,
                            },
                            {
                                id: "max",
                                header: "Max %",
                                align: "right",
                                cell: (r) => `${r.max_percentage}%`,
                            },
                        ]}
                        data={profile.ranges}
                        getRowId={(r) => r.id}
                    />
                </div>
            )}

            {/* Edit ranges */}
            <div className="space-y-2">
                <h2 className="font-medium">
                    {profile.ranges && profile.ranges.length > 0 ? "Update Ranges" : "Set Ranges"}
                </h2>
                <SetScaleRangesForm profileId={profileId} />
            </div>
        </article>
    );
}
