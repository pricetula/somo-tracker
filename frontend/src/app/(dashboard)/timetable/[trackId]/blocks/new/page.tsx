/**
 * Add Time Block Page — /timetable/[trackId]/blocks/new
 */
"use client";

import { useParams } from "next/navigation";
import { BlockCreateForm } from "@/features/timetable/components/block-create-form";

export default function BlockNewPage() {
    const params = useParams();
    const trackId = params?.trackId as string;

    return (
        <div className="p-6">
            <div className="mx-auto max-w-2xl space-y-6">
                <div>
                    <h1 className="text-xl font-semibold">Add Time Block</h1>
                    <p className="text-muted-foreground text-sm">
                        Define a new period for this timetable.
                    </p>
                </div>
                <BlockCreateForm trackId={trackId} />
            </div>
        </div>
    );
}
