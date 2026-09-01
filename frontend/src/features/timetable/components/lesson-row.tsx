import React from "react";
import Link from "next/link";
import { Clock } from "lucide-react";
import { format, parseISO, isBefore, isAfter } from "date-fns";
import { motion } from "framer-motion";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { LessonEntry } from "../hooks";

export type LessonStatus = "past" | "present" | "future";

export function getLessonStatus(start: Date | null, end: Date | null): LessonStatus {
    if (!start || !end) return "future";
    const now = new Date();
    if (isBefore(now, start)) return "future";
    if (isAfter(now, end)) return "past";
    return "present";
}

export interface LessonRowProps {
    entry: LessonEntry;
    isFirst: boolean;
    isLast: boolean;
    isScrollTarget?: boolean;
}

export function LessonRow({ entry, isLast, isScrollTarget }: LessonRowProps) {
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
        <motion.article
            id={isScrollTarget ? "timeline-scroll-target" : undefined}
            initial={isScrollTarget ? { opacity: 0, scale: 0.97 } : false}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ duration: 0.4, ease: "easeOut" }}
            className="relative flex gap-1"
        >
            <aside className="sticky top-0 w-30 shrink-0 space-y-2 self-start pt-5 pr-2 pb-4 text-right">
                <div className="text-foreground leading-tight font-medium">{dateLabel}</div>
                <div className="text-muted-foreground text-[10px] leading-tight tracking-wide">
                    {timeRange}
                </div>
            </aside>

            <div className="relative flex w-10 shrink-0 flex-col items-center pt-5 pb-4">
                <Tooltip>
                    <TooltipTrigger>
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
                    </TooltipTrigger>
                    <TooltipContent className="side-top max-w-xs p-3 text-xs">
                        {isBreak
                            ? "Non instructional period " + entry?.period_name
                            : `${entry?.period_name ?? "lesson"} ${
                                  (status === "past" && "has finished") ||
                                  (status === "present" && "is ongoing") ||
                                  (status === "future" && "is upcoming")
                              }`}
                    </TooltipContent>
                </Tooltip>
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
                    <Link href={`/classes/${entry.class_id}`}>{entry.class_name}</Link>
                </div>
            </div>
        </motion.article>
    );
}
