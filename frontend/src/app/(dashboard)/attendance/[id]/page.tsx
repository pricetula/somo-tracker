/**
 * Allocation Page — /timetable/[trackId]/allocate
 */
"use client";

import { Suspense } from "react";
import { AllocationForm } from "@/features/timetable/components/allocation-form";

function AllocateContent() {
    return (
        <div className="p-6">
            <div className="mx-auto max-w-lg">
                <h1 className="mb-1 text-lg font-semibold">Assign Teacher</h1>
                <p className="text-muted-foreground mb-6">
                    Select a learning area and teacher to assign to this timetable slot.
                </p>
                <AllocationForm />
            </div>
        </div>
    );
}

export default function Page() {
    return (
        <Suspense
            fallback={
                <div className="p-6">
                    <div className="mx-auto max-w-lg space-y-4">
                        <div className="bg-muted h-6 w-40 animate-pulse rounded" />
                        <div className="bg-muted h-4 w-72 animate-pulse rounded" />
                        <div className="bg-muted h-10 w-full animate-pulse rounded" />
                    </div>
                </div>
            }
        >
            <AllocateContent />
        </Suspense>
    );
}
