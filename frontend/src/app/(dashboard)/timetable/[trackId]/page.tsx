/**
 * Track Detail Page (route handler) — fetches trackId from URL and delegates to feature.
 */
"use client";

import { useParams } from "next/navigation";
import { TrackDetail } from "@/features/timetable/components/track-detail";

export default function TrackDetailPage() {
    const params = useParams();
    const trackId = (params?.trackId as string) ?? "";
    return <TrackDetail trackId={trackId} />;
}
