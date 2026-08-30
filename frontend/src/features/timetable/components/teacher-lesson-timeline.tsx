/**
 * Teacher Lesson Timeline — redesigned to match shadcnstudio timeline-05.
 * Sticky date labels left, center vertical rail with dots, flat content right,
 * expandable accordions, semantic CSS variables, no hard-coded hex colors.
 */
"use client";

import React, { useCallback } from "react";
import { format, parseISO } from "date-fns";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import {
    Accordion,
    AccordionContent,
    AccordionItem,
    AccordionTrigger,
} from "@/components/ui/accordion";
import { useTeacherLessonTimeline, type LessonEntry, type LessonTimelinePage } from "../hooks";
import { getErrorMessage } from "@/lib/errors";

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

    if (isLoading && !pages.length) return <LessonTimelineSkeleton />;
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
            className="bg-background max-h-95 overflow-y-auto scroll-smooth rounded-md"
            onScroll={handleScroll}
        >
            <div className="space-y-0">
                {entries.map((entry, index) => (
                    <TimelineRow
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

/* ─── Timeline Row ───────────────────────────────────────────────────── */

function TimelineRow({
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
    const isBreak = !!entry.is_break;

    return (
        <article className="relative flex gap-0 md:gap-2 lg:gap-4">
            {/* Left sticky label column */}
            <aside className="sticky top-0 w-24 shrink-0 self-start pt-5 pr-2 pb-4 text-right md:pr-3">
                <div className="text-foreground block leading-tight font-medium">{dateLabel}</div>
                <div className="text-muted-foreground block text-[10px] leading-tight tracking-wide">
                    {timeRange}
                </div>
            </aside>

            {/* Center rail */}
            <div className="relative flex w-10 shrink-0 flex-col items-center pt-5 pb-4">
                {!isFirst && <div className="bg-muted-foreground/20 h-3 w-px" />}
                {isFirst && <div className="h-3" />}
                <div className="relative z-10">
                    <span
                        className={
                            "ring-background block h-3 w-3 rounded-full ring-2 " +
                            (isBreak ? "bg-muted-foreground" : "bg-primary")
                        }
                    />
                </div>
                {!isLast && <div className="bg-muted-foreground/20 min-h-8 w-px flex-1" />}
            </div>

            {/* Right content */}
            <div className="min-w-0 flex-1 space-y-3 pt-5 pr-4 pb-6 md:pr-6 lg:pl-2">
                {/* Header */}
                <div className="space-y-1">
                    <div className="flex flex-wrap items-center gap-2">
                        <h3 className="text-foreground text-xl leading-snug font-semibold">
                            {entry.class_name}
                        </h3>
                        <Badge variant="outline" className="h-5 px-1.5 text-[0.625rem]">
                            {entry.subject_name}
                        </Badge>
                    </div>
                    <p className="text-muted-foreground text-sm leading-relaxed">
                        {entry.period_name}
                        {entry.room ? ` · ${entry.room}` : ""}
                        {isBreak ? " — Break / Prep" : ""}
                    </p>
                </div>

                {/* Details list */}
                <ul className="text-muted-foreground ml-2 list-inside list-disc space-y-1.5 text-sm leading-relaxed">
                    <li>
                        <span className="text-foreground font-medium">Start:</span>{" "}
                        {format(start, "h:mm a")}
                    </li>
                    <li>
                        <span className="text-foreground font-medium">End:</span>{" "}
                        {format(end, "h:mm a")}
                    </li>
                    <li>
                        <span className="text-foreground font-medium">Subject:</span>{" "}
                        {entry.subject_name}
                    </li>
                </ul>

                {/* Tags */}
                <div className="flex flex-wrap items-center gap-2">
                    <Badge variant="secondary" className="h-6 rounded-sm">
                        {entry.period_name}
                    </Badge>
                    {entry.room && (
                        <Badge variant="outline" className="h-6 rounded-sm">
                            {entry.room}
                        </Badge>
                    )}
                    {isBreak ? (
                        <Badge className="bg-muted-foreground/10 text-muted-foreground h-6 rounded-sm">
                            Break
                        </Badge>
                    ) : (
                        <Badge className="h-6 rounded-sm bg-emerald-600/10 text-emerald-600">
                            Active
                        </Badge>
                    )}
                </div>

                {/* Expandable details — reference accordion style */}
                <Accordion defaultValue={["details"]} className="w-full border-none">
                    <AccordionItem value="details" className="border-none bg-transparent">
                        <AccordionTrigger className="hover:text-foreground text-muted-foreground gap-2 px-0 py-1 font-medium hover:no-underline">
                            <span className="inline-flex items-center gap-2">
                                <span className="flex h-5 w-5 items-center justify-center rounded-sm bg-emerald-600/10 px-1 text-[10px] font-bold text-emerald-600">
                                    New
                                </span>
                                Lesson details
                            </span>
                        </AccordionTrigger>
                        <AccordionContent>
                            <div className="text-muted-foreground space-y-2 pt-2 pb-1 pl-1 text-sm leading-relaxed">
                                <p>
                                    {isBreak
                                        ? "Break period — use for prep, grading, or student support."
                                        : `Scheduled lesson for ${entry.class_name}. Focus on ${entry.subject_name} during the ${entry.period_name} block.`}
                                </p>
                                <ul className="ml-1 list-inside list-disc space-y-1">
                                    <li>Review prior week materials</li>
                                    <li>Prepare assessment for next session</li>
                                    <li>Update attendance records</li>
                                </ul>
                            </div>
                        </AccordionContent>
                    </AccordionItem>
                    {!isBreak && (
                        <AccordionItem value="updates" className="border-none bg-transparent">
                            <AccordionTrigger className="hover:text-foreground text-muted-foreground gap-2 px-0 py-1 font-medium hover:no-underline">
                                <span className="inline-flex items-center gap-2">
                                    <span className="bg-primary/10 text-primary flex h-5 w-5 items-center justify-center rounded-sm px-1 text-[10px] font-bold">
                                        U
                                    </span>
                                    Updates
                                </span>
                            </AccordionTrigger>
                            <AccordionContent>
                                <div className="text-muted-foreground space-y-2 pt-2 pb-1 pl-1 text-sm leading-relaxed">
                                    <p>Room allocation updated for this session.</p>
                                    <ul className="ml-1 list-inside list-disc space-y-1">
                                        <li>Confirm tech setup before start</li>
                                        <li>Share resources link with students</li>
                                    </ul>
                                </div>
                            </AccordionContent>
                        </AccordionItem>
                    )}
                </Accordion>
            </div>
        </article>
    );
}

/* ─── Skeleton ─────────────────────────────────────────────────────────── */

function LessonTimelineSkeleton() {
    return (
        <div className="bg-background space-y-0">
            {Array.from({ length: 4 }).map((_, i) => (
                <div key={i} className="relative flex gap-0 py-5 md:gap-2 lg:gap-4">
                    <aside className="sticky top-0 w-24 shrink-0 self-start px-2 pt-5 text-right md:w-32 md:px-3 lg:w-36">
                        <Skeleton className="mb-1 ml-auto h-4 w-16" />
                        <Skeleton className="ml-auto h-3 w-12" />
                    </aside>
                    <div className="flex w-10 shrink-0 flex-col items-center pt-5">
                        <Skeleton className="h-3 w-px" />
                        <Skeleton className="h-3 w-3 rounded-full" />
                        <Skeleton className="h-12 w-px" />
                    </div>
                    <div className="min-w-0 flex-1 space-y-3 pt-5 pr-4">
                        <div className="flex items-center gap-3">
                            <Skeleton className="h-6 w-40" />
                            <Skeleton className="h-5 w-16 rounded-full" />
                        </div>
                        <Skeleton className="h-4 w-3/4" />
                        <div className="flex gap-2">
                            <Skeleton className="h-5 w-20 rounded-full" />
                            <Skeleton className="h-5 w-14 rounded-full" />
                        </div>
                    </div>
                </div>
            ))}
        </div>
    );
}
