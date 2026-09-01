/**
 * Teacher Lesson Timeline — redesigned to match shadcnstudio timeline-05.
 * Sticky date labels left, center vertical rail with dots, flat content right,
 * expandable accordions, semantic CSS variables, no hard-coded hex colors.
 */
"use client";

import React, { useCallback, useEffect, useRef } from "react";
import { useTeacherLessonTimeline, type LessonEntry, type LessonTimelinePage } from "../hooks";
import { getErrorMessage } from "@/lib/errors";
import { LessonRow, getLessonStatus } from "./lesson-row";
import { LessonTimelineSkeleton } from "./lesson-timeline-skeleton";
import { parseISO } from "date-fns";

interface TeacherLessonTimelineProps {
    teacherId: string;
}

export function TeacherLessonTimeline({ teacherId }: TeacherLessonTimelineProps) {
    const { data, isLoading, isFetchingNextPage, hasNextPage, fetchNextPage, isError, error } =
        useTeacherLessonTimeline({ teacherId });

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

    const hasScrolled = useRef(false);
    const targetIndex = React.useMemo(() => {
        let firstFuture = -1;
        for (let i = 0; i < entries.length; i++) {
            const entry = entries[i];
            const start = entry?.start_time ? parseISO(entry.start_time) : null;
            const end = entry?.end_time ? parseISO(entry.end_time) : null;
            const status = getLessonStatus(start, end);
            if (status === "present") return i;
            if (status === "future" && firstFuture === -1) firstFuture = i;
        }
        return firstFuture;
    }, [entries]);

    useEffect(() => {
        if (targetIndex >= 0 && !hasScrolled.current) {
            const el = document.getElementById("timeline-scroll-target");
            if (el) {
                el.scrollIntoView({ block: "start", behavior: "smooth" });
                hasScrolled.current = true;
            }
        }
    }, [targetIndex, entries.length]);

    if (isLoading && !pages.length) return <LessonTimelineSkeleton />;
    if (isError && error) {
        return (
            <div className="bg-background rounded-md p-6">
                <p className="text-destructive font-medium">{getErrorMessage(error)}</p>
            </div>
        );
    }
    if (!entries.length) {
        return (
            <div className="bg-background flex items-center justify-center p-8">
                <p className="text-muted-foreground">No lessons found for this week.</p>
            </div>
        );
    }

    return (
        <div
            className="bg-background max-h-95 overflow-y-auto scroll-smooth rounded-md"
            onScroll={handleScroll}
        >
            <div className="space-y-0">
                {entries.map((entry, index) => (
                    <LessonRow
                        key={`${entry.id}-${index}`}
                        entry={entry}
                        isFirst={index === 0}
                        isLast={index === entries.length - 1 && !hasNextPage}
                        isScrollTarget={index === targetIndex}
                    />
                ))}
                {isFetchingNextPage && (
                    <div className="px-4 py-4">
                        <div className="bg-muted h-16 w-full animate-pulse rounded-md" />
                    </div>
                )}
            </div>
        </div>
    );
}
