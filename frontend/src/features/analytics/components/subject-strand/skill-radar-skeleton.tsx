"use client";

export function SkillRadarSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-32 animate-pulse rounded" />
            <div className="bg-muted mx-auto aspect-square max-h-[320px] w-full animate-pulse rounded" />
        </div>
    );
}
