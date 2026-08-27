/**
 * Track Detail Page — /timetable/[trackId]
 * Shows the block grid for one timetable track.
 */
"use client";

import React from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { ArrowLeft, PlusIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { TimeTable } from "@/features/timetable";

export default function TrackDetailPage() {
    const params = useParams();
    const router = useRouter();
    const trackId = params?.trackId as string;

    return (
        <div className="space-y-4">
            <div className="flex items-center gap-3">
                <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => router.push("/timetable")}
                    aria-label="Back"
                >
                    <ArrowLeft className="size-4" />
                </Button>
                <div className="flex-1">
                    <h1 className="text-xl font-semibold tracking-tight">Track {trackId}</h1>
                    <p className="text-muted-foreground text-sm">Time blocks and assignments</p>
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
