/**
 * Track Detail — loads timetable data for one track by ID.
 */
"use client";

import React from "react";
import { PlusIcon } from "lucide-react";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useTrack } from "../hooks/use-timetable";
import { TimeTable } from "./timetable";

export function TrackDetail({ trackId }: { trackId: string }) {
    const { data: track, isLoading: trackLoading } = useTrack(trackId);

    if (trackLoading) {
        return (
            <div className="space-y-4 p-6">
                <div className="flex items-center gap-3">
                    <Skeleton className="h-6 w-48" />
                </div>
                <Skeleton className="h-96 w-full rounded-lg" />
            </div>
        );
    }

    return (
        <div className="space-y-4">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-xl font-semibold tracking-tight">
                        {track?.name ?? "Track"}
                    </h1>
                    {track?.description && (
                        <p className="text-muted-foreground text-sm">{track.description}</p>
                    )}
                </div>
                <Link href={`/timetable/${trackId}/blocks/new`}>
                    <Button size="sm">
                        <PlusIcon className="mr-1.5 size-3.5" />
                        Add Time Block
                    </Button>
                </Link>
            </div>
            <TimeTable trackId={trackId} />
        </div>
    );
}
