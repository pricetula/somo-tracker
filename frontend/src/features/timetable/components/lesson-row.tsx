import React from "react";
import { format, parseISO } from "date-fns";
import { Badge } from "@/components/ui/badge";
import {
    Accordion,
    AccordionContent,
    AccordionItem,
    AccordionTrigger,
} from "@/components/ui/accordion";
import type { LessonEntry } from "../hooks";

export interface LessonRowProps {
    entry: LessonEntry;
    isFirst: boolean;
    isLast: boolean;
}

export function LessonRow({ entry, isLast }: LessonRowProps) {
    const start = parseISO(entry.start_time);
    const end = parseISO(entry.end_time);
    const dateLabel = format(start, "EEE MMM d");
    const timeRange = `${format(start, "h:mm a")} — ${format(end, "h:mm a")}`;
    const isBreak = !!entry.is_break;

    return (
        <article className="relative flex gap-1">
            <aside className="sticky top-0 w-30 shrink-0 self-start pt-5 pr-2 pb-4 text-right">
                <div className="text-foreground leading-tight font-medium">{dateLabel}</div>
                <div className="text-muted-foreground text-[10px] leading-tight tracking-wide">
                    {timeRange}
                </div>
            </aside>

            <div className="relative flex w-10 shrink-0 flex-col items-center pt-5 pb-4">
                {/*{!isFirst && <div className="bg-muted-foreground/20 h-3 w-px" />}
                {isFirst && <div className="h-3" />}*/}
                <span
                    className={
                        "ring-background z-10 h-3 w-3 rounded-full ring-2 " +
                        (isBreak ? "bg-muted-foreground" : "bg-primary")
                    }
                />
                {!isLast && <div className="bg-muted-foreground/20 min-h-8 w-px flex-1" />}
            </div>

            <div className="min-w-0 flex-1 space-y-3 pt-5 pr-4 pb-6 md:pr-6 lg:pl-2">
                <div className="space-y-1">
                    <div className="flex flex-wrap items-center gap-2">
                        <h3 className="text-foreground text-xl leading-snug font-semibold">
                            {entry.class_name}
                        </h3>
                        <Badge variant="outline" className="h-5 px-1.5 text-[0.625rem]">
                            {entry.subject_name}
                        </Badge>
                    </div>
                    <p className="text-muted-foreground leading-relaxed">
                        {entry.period_name}
                        {entry.room ? ` · ${entry.room}` : ""}
                        {isBreak ? " — Break / Prep" : ""}
                    </p>
                </div>

                <ul className="text-muted-foreground ml-2 list-inside list-disc space-y-1.5 leading-relaxed">
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

                <Accordion defaultValue={["details"]} className="w-full border-none">
                    <AccordionItem value="details" className="border-none bg-transparent">
                        <AccordionTrigger className="text-muted-foreground hover:text-foreground gap-2 px-0 py-1 font-medium hover:no-underline">
                            <span className="inline-flex items-center gap-2">
                                <span className="flex h-5 w-5 items-center justify-center rounded-sm bg-emerald-600/10 px-1 font-bold text-emerald-600">
                                    New
                                </span>
                                Lesson details
                            </span>
                        </AccordionTrigger>
                        <AccordionContent>
                            <div className="text-muted-foreground space-y-2 pt-2 pb-1 pl-1 leading-relaxed">
                                <p>
                                    {isBreak
                                        ? "Break period — use for prep, grading, or student support."
                                        : `Scheduled lesson for ${entry.class_name}. Focus on ${entry.subject_name} during the ${entry.period_name} .`}
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
                            <AccordionTrigger className="text-muted-foreground hover:text-foreground gap-2 px-0 py-1 font-medium hover:no-underline">
                                <span className="inline-flex items-center gap-2">
                                    <span className="bg-primary/10 text-primary flex h-5 w-5 items-center justify-center rounded-sm px-1 font-bold">
                                        U
                                    </span>
                                    Updates
                                </span>
                            </AccordionTrigger>
                            <AccordionContent>
                                <div className="text-muted-foreground space-y-2 pt-2 pb-1 pl-1 leading-relaxed">
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
