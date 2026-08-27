/**
 * Add Time Block Page — /timetable/[trackId]/blocks/new
 */
"use client";

import { useParams, useRouter } from "next/navigation";
import { useCallback } from "react";
import { ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { BlockCreateForm } from "@/features/timetable/components/block-create-form";

export default function BlockNewPage() {
    const params = useParams();
    const router = useRouter();
    const trackId = params?.trackId as string;

    const handleSuccess = useCallback(() => {
        router.push(`/timetable/${trackId}`);
    }, [router, trackId]);

    const handleBack = useCallback(() => {
        router.push(`/timetable/${trackId}`);
    }, [router, trackId]);

    return (
        <div className="p-6">
            <div className="mx-auto max-w-2xl space-y-6">
                <div className="flex items-center gap-3">
                    <Button variant="ghost" size="icon" onClick={handleBack} aria-label="Back">
                        <ArrowLeft className="size-4" />
                    </Button>
                    <div>
                        <h1 className="text-xl font-semibold">Add Time Block</h1>
                        <p className="text-muted-foreground text-sm">
                            Define a new period for this timetable.
                        </p>
                    </div>
                </div>
                <BlockCreateForm trackId={trackId} />
            </div>
        </div>
    );
}
