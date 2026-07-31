/**
 * PercentileGaugeSkeleton — Loading and error skeletons for PercentileGauge.
 *
 * Visualisation: Circular progress rings with placeholder colors while data loads.
 * Props: None.
 */
"use client";

export function PercentileGaugeSkeleton() {
    return (
        <div className="space-y-4">
            <div className="bg-muted h-4 w-40 animate-pulse rounded" />
            <div className="grid grid-cols-2 gap-4">
                <div className="bg-muted/30 animate-pulse rounded-md p-4">
                    <div className="bg-muted mx-auto h-3 w-16 rounded" />
                    <div className="bg-muted mx-auto mt-2 h-10 w-20 rounded" />
                </div>
                <div className="bg-muted/30 animate-pulse rounded-md p-4">
                    <div className="bg-muted mx-auto h-3 w-16 rounded" />
                    <div className="bg-muted mx-auto mt-2 h-10 w-20 rounded" />
                </div>
            </div>
        </div>
    );
}
