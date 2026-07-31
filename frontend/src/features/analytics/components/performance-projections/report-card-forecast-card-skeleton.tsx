"use client";

export function ReportCardForecastCardSkeleton() {
    return (
        <div className="bg-muted/20 animate-pulse space-y-3 rounded-md p-4">
            <div className="bg-muted h-4 w-36 rounded" />
            <div className="bg-muted mx-auto h-10 w-24 rounded" />
            <div className="bg-muted mx-auto h-4 w-32 rounded" />
        </div>
    );
}
