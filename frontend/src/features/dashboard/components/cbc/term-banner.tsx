"use client";

import { useMemo } from "react";
import { AlertTriangle } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import { Button } from "@/components/ui/button";
import type { TermBannerData } from "./types";
import { termWeekInfo } from "./data";

function computeProgress(startDate: string, endDate: string): number {
    const now = new Date().getTime();
    const start = new Date(startDate).getTime();
    const end = new Date(endDate).getTime();
    if (now <= start) return 0;
    if (now >= end) return 100;
    return Math.round(((now - start) / (end - start)) * 100);
}

function AmberIcon() {
    return <AlertTriangle className="mr-1.5 inline size-4 text-amber-500" />;
}

export function TermBanner({ data }: { data: TermBannerData }) {
    const progress = useMemo(() => {
        if (data.state === "active") {
            return computeProgress(data.startDate, data.endDate);
        }
        return 0;
    }, [data]);

    const renderContent = () => {
        switch (data.state) {
            case "active":
                return (
                    <div className="flex flex-col gap-1.5">
                        <span className="text-sm font-medium">
                            {data.termName} / {data.yearName}
                        </span>
                        <Progress value={progress} className="h-2" />
                        <span className="text-muted-foreground text-xs">
                            Week {termWeekInfo.currentWeek} of {termWeekInfo.totalWeeks}
                        </span>
                    </div>
                );

            case "no-term":
                return (
                    <div className="flex flex-col gap-1.5">
                        <span className="text-sm font-medium">
                            <AmberIcon />
                            {data.yearName} / No active term
                        </span>
                        <Progress value={0} className="h-2 opacity-40" />
                        <HoverCard>
                            <HoverCardTrigger asChild>
                                <span className="text-muted-foreground cursor-help text-xs underline decoration-dotted underline-offset-2">
                                    Why is data entry restricted?
                                </span>
                            </HoverCardTrigger>
                            <HoverCardContent className="w-72 text-xs">
                                <p className="mb-2">
                                    The current date falls outside any configured term dates. Data
                                    entry is restricted.
                                </p>
                                <Button variant="link" size="sm" className="h-auto p-0 text-xs">
                                    Create new term period →
                                </Button>
                            </HoverCardContent>
                        </HoverCard>
                    </div>
                );

            case "no-config":
                return (
                    <div className="flex flex-col gap-1.5">
                        <span className="text-sm font-medium">
                            <AmberIcon />
                            System initialization required
                        </span>
                        <HoverCard>
                            <HoverCardTrigger asChild>
                                <span className="text-muted-foreground cursor-help text-xs underline decoration-dotted underline-offset-2">
                                    Why do I need to configure this?
                                </span>
                            </HoverCardTrigger>
                            <HoverCardContent className="w-72 text-xs">
                                <p className="mb-2">
                                    No academic year or term data exists. You cannot register
                                    students, log attendance, or issue invoices without a time
                                    anchor.
                                </p>
                                <Button variant="link" size="sm" className="h-auto p-0 text-xs">
                                    Configure academic year →
                                </Button>
                            </HoverCardContent>
                        </HoverCard>
                    </div>
                );

            case "out-of-range":
                return (
                    <div className="flex flex-col gap-1.5">
                        <span className="text-sm font-medium">
                            <AmberIcon />
                            Calendar date out of range
                        </span>
                        <HoverCard>
                            <HoverCardTrigger asChild>
                                <span className="text-muted-foreground cursor-help text-xs underline decoration-dotted underline-offset-2">
                                    What does this mean?
                                </span>
                            </HoverCardTrigger>
                            <HoverCardContent className="w-72 text-xs">
                                <p className="mb-2">
                                    The system clock detects today&apos;s date is outside your
                                    active structural calendar boundaries.
                                </p>
                                <Button variant="link" size="sm" className="h-auto p-0 text-xs">
                                    View calendar management →
                                </Button>
                            </HoverCardContent>
                        </HoverCard>
                    </div>
                );
        }
    };

    return (
        <Card>
            <CardContent>{renderContent()}</CardContent>
        </Card>
    );
}
