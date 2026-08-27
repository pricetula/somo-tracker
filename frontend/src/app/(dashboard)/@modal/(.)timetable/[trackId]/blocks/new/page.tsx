/**
 * Intercepted add-blocks — modal overlay from track detail.
 */
"use client";
import { useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
} from "@/components/ui/dialog";
import { BlockCreateForm } from "@/features/timetable/components/block-create-form";

export default function BlockNewModal() {
    const router = useRouter();
    const params = useParams();
    const trackId = (params?.trackId as string) ?? "";
    const handleBack = useCallback(() => router.back(), [router]);

    return (
        <Dialog open onOpenChange={handleBack}>
            <DialogContent className="max-h-[90vh] w-full overflow-y-auto sm:max-w-2xl">
                <DialogHeader>
                    <DialogTitle>Add Time Block</DialogTitle>
                    <DialogDescription>Define a new period for this timetable.</DialogDescription>
                </DialogHeader>
                <BlockCreateForm trackId={trackId} />
            </DialogContent>
        </Dialog>
    );
}
