/**
 * Teacher Lesson Timeline — flat vertical timeline for a teacher's weekly lessons.
 * Infinite scroll pagination, flat background, no cards, date-fns formatting.
 */
"use client";

import React, { useCallback } from "react";
import { format, parseISO } from "date-fns";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { useTeacherLessonTimeline, type LessonEntry, type LessonTimelinePage } from "../hooks";
import { getErrorMessage } from "@/lib/errors";

// ─── Component ────────────────────────────────────────────────────────────

interface TeacherLessonTimelineProps {
    teacherId: string;
}

export function TeacherLessonTimeline({ teacherId }: TeacherLessonTimelineProps) {
    const { data, isLoading, isFetchingNextPage, hasNextPage, fetchNextPage, isError, error } =
        useTeacherLessonTimeline({ teacherId });

    // Load more on scroll near bottom
    const handleScroll = useCallback(
        (e: React.UIEvent<HTMLDivElement>) => {
            const el = e.currentTarget;
            const nearBottom = el.scrollTop + el.clientHeight >= el.scrollHeight - 80;
            if (nearBottom && hasNextPage && !isFetchingNextPage) {
                fetchNextPage();
            }
        },
        [hasNextPage, isFetchingNextPage, fetchNextPage]
    );

    const pages = (data?.pages as LessonTimelinePage[] | undefined) ?? [];
    const entries: LessonEntry[] = pages.flatMap((p: LessonTimelinePage) => p.entries ?? []);

    if (isLoading && !pages.length) {
        return <LessonTimelineSkeleton />;
    }

    if (isError && error) {
        return (
            <div className="bg-background rounded-md p-6">
                <p className="text-destructive text-sm font-medium">{getErrorMessage(error)}</p>
            </div>
        );
    }

    if (!entries.length) {
        return (
            <div className="bg-background flex items-center justify-center p-8">
                <p className="text-muted-foreground text-sm">No lessons found for this week.</p>
            </div>
        );
    }

    return (
        <div
            className="bg-background max-h-[70vh] overflow-y-auto scroll-smooth rounded-md"
            onScroll={handleScroll}
        >
            <div className="space-y-0">
                {entries.map((entry, index) => (
                    <LessonRow
                        key={`${entry.id}-${index}`}
                        entry={entry}
                        isFirst={index === 0}
                        isLast={index === entries.length - 1 && !hasNextPage}
                    />
                ))}
                {isFetchingNextPage && (
                    <div className="px-4 py-4">
                        <Skeleton className="h-16 w-full rounded-md" />
                    </div>
                )}
            </div>
        </div>
    );
}

// ─── Row Component ───────────────────────────────────────────────────────

function LessonRow({
    entry,
    isFirst,
    isLast,
}: {
    entry: LessonEntry;
    isFirst: boolean;
    isLast: boolean;
}) {
    const start = parseISO(entry.start_time);
    const end = parseISO(entry.end_time);
    const dateLabel = format(start, "EEE MMM d");
    const timeRange = `${format(start, "h:mm a")} — ${format(end, "h:mm a")}`;

    return (
        <article className="bg-background relative flex gap-4 px-4 py-5">
            {/* Timeline node + connector */}
            <div className="relative flex w-8 shrink-0 flex-col items-center">
                {/* Top connector */}
                {!isFirst && <div className="bg-muted-foreground/20 h-3 w-px" />}
                {isFirst && <div className="h-3" />}
                {/* Node */}
                <div className="bg-primary ring-background z-10 h-3 w-3 rounded-full ring-2" />
                {/* Bottom connector */}
                {!isLast && <div className="bg-muted-foreground/20 min-h-8 w-px flex-1" />}
            </div>

            {/* Content */}
            <div className="min-w-0 flex-1 space-y-1.5">
                <div className="flex flex-wrap items-center gap-2">
                    <Badge variant="default" className="text-[0.625rem]">
                        {entry.subject_name}
                    </Badge>
                    <span className="text-muted-foreground text-[10px] tracking-wide uppercase">
                        {entry.period_name}
                    </span>
                </div>
                <h3 className="text-foreground truncate text-sm leading-snug font-medium">
                    {entry.class_name}
                </h3>
                <div className="text-muted-foreground flex items-center gap-3 text-[11px]">
                    <time dateTime={entry.start_time}>
                        {dateLabel} · {timeRange}
                    </time>
                    {entry.room && <span className="truncate">{entry.room}</span>}
                </div>
            </div>
        </article>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────

function LessonTimelineSkeleton() {
    return (
        <div className="bg-background space-y-0">
            {Array.from({ length: 4 }).map((_, i) => (
                <div key={i} className="relative flex gap-4 px-4 py-5">
                    <div className="relative flex w-8 shrink-0 flex-col items-center">
                        <div className="bg-muted-foreground/20 h-3 w-px" />
                        <Skeleton className="h-3 w-3 rounded-full" />
                        <div className="bg-muted-foreground/20 min-h-8 w-px flex-1" />
                    </div>
                    <div className="flex-1 space-y-2">
                        <div className="flex gap-2">
                            <Skeleton className="h-4 w-16 rounded-full" />
                            <Skeleton className="h-3 w-12 rounded-full" />
                        </div>
                        <Skeleton className="h-4 w-32" />
                        <Skeleton className="h-3 w-24" />
                    </div>
                </div>
            ))}
        </div>
    );
}
