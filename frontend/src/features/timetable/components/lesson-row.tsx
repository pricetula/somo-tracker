import React, { useMemo } from "react";
import { Clock } from "lucide-react";
import { format, parseISO, isBefore, isAfter } from "date-fns";
import Link from "next/link";
import type { LessonEntry } from "../hooks";

export type LessonStatus = "past" | "present" | "future";

function getLessonStatus(start: Date, end: Date): LessonStatus {
    const now = new Date();
    if (isBefore(now, start)) return "future";
    if (isAfter(now, end)) return "past";
    return "present";
}

export interface LessonRowProps {
    entry: LessonEntry;
    isFirst: boolean;
    isLast: boolean;
}

export function LessonRow({ entry, isLast }: LessonRowProps) {
    const { start, end } = React.useMemo(
        () => ({
            start: entry?.start_time ? parseISO(entry.start_time) : null,
            end: entry?.end_time ? parseISO(entry.end_time) : null,
        }),
        [entry]
    );
    const dateLabel = React.useMemo(() => format(start, "EEE MMM d"), [start]);
    const timeRange = React.useMemo(
        () => `${format(start, "h:mm a")} — ${format(end, "h:mm a")}`,
        [start, end]
    );
    const isBreak = !!entry.is_break;
    const status = React.useMemo(() => getLessonStatus(start, end), [start, end]);

    return (
        <article className="relative flex gap-1">
            <aside className="sticky top-0 w-30 shrink-0 space-y-2 self-start pt-5 pr-2 pb-4 text-right">
                <div className="text-foreground leading-tight font-medium">{dateLabel}</div>
                <div className="text-muted-foreground text-[10px] leading-tight tracking-wide">
                    {timeRange}
                </div>
            </aside>

            <div className="relative flex w-10 shrink-0 flex-col items-center pt-5 pb-4">
                <Clock
                    size={16}
                    className={
                        isBreak
                            ? "text-muted-foreground"
                            : status === "past"
                              ? "text-muted-foreground/60"
                              : status === "present"
                                ? "text-emerald-500"
                                : "text-primary"
                    }
                />
                {!isLast && <div className="bg-muted-foreground/20 min-h-8 w-px flex-1" />}
            </div>

            <div className="min-w-0 flex-1 space-y-3 pt-3 pr-4 pb-6 md:pr-6 lg:pl-2">
                <div className="space-y-1">
                    <h3 className="text-foreground mb-6 text-base font-semibold">
                        <Link href={`/curriculum/${entry.subject_id}`}>{entry.subject_name}</Link>
                    </h3>
                    <p className="text-muted-foreground leading-relaxed">
                        {entry.period_name}
                        {entry.room ? ` · ${entry.room}` : ""}
                        {isBreak ? " — Break / Prep" : ""}
                    </p>
                    {!isBreak && (
                        <span
                            className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${
                                status === "past"
                                    ? "bg-muted text-muted-foreground"
                                    : status === "present"
                                      ? "bg-emerald-100 text-emerald-700"
                                      : "bg-blue-100 text-blue-700"
                            }`}
                        >
                            {status === "past"
                                ? "Ended"
                                : status === "present"
                                  ? "Ongoing"
                                  : "Upcoming"}
                        </span>
                    )}
                    <Link href={`/classes/${entry.class_id}`}>{entry.class_name}</Link>
                </div>
            </div>
        </article>
    );
}
