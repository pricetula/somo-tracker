"use client";

import { format } from "date-fns";
import { Clock, PlayCircle, CheckCircle2, AlertCircle, Coffee } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { type EnrichedSlot } from "@/lib/api/timetable-structure";
import Link from "next/link";
import * as React from "react";

type SlotStatus = "active" | "missed" | "completed" | "upcoming" | "break";

function formatTime(timeStr: string) {
    // Convert "HH:MM:SS" or "HH:MM" to 12-hour format
    try {
        return format(new Date(`2000-01-01T${timeStr}`), "h:mm a");
    } catch {
        return timeStr;
    }
}

const statusConfig: Record<SlotStatus, { icon: React.ReactNode; label: string; color: string }> = {
    active: {
        icon: <PlayCircle className="size-4" />,
        label: "Active",
        color: "text-emerald-600",
    },
    missed: {
        icon: <AlertCircle className="size-4" />,
        label: "Missed",
        color: "text-destructive",
    },
    completed: {
        icon: <CheckCircle2 className="size-4" />,
        label: "Completed",
        color: "text-blue-600",
    },
    upcoming: {
        icon: <Clock className="size-4" />,
        label: "Upcoming",
        color: "text-amber-600",
    },
    break: {
        icon: <Coffee className="size-4" />,
        label: "Break",
        color: "text-muted-foreground",
    },
};

export function TimelineItem({
    slot,
    status,
    date,
}: {
    slot: EnrichedSlot;
    status: SlotStatus;
    date: string;
}) {
    const cfg = statusConfig[status];

    if (status === "break") {
        return (
            <div className="bg-muted/20 text-muted-foreground flex items-center gap-3 px-4 py-2 text-sm">
                <Coffee className="size-4 shrink-0" />
                <span className="font-medium">{slot.period_name}</span>
                <span className="text-xs">
                    {formatTime(slot.start_time)} – {formatTime(slot.end_time)}
                </span>
            </div>
        );
    }

    return (
        <div
            className={`group relative border-l-2 px-4 py-3 ${
                status === "active"
                    ? "border-l-green-500 bg-green-600/5"
                    : status === "missed"
                      ? "border-l-red-500 bg-red-600/5"
                      : status === "completed"
                        ? "border-l-blue-500 bg-blue-600/5"
                        : status === "upcoming"
                          ? "border-l-amber-500 bg-amber-600/5"
                          : "bg-muted/30 border-l-transparent"
            }`}
        >
            <div className="flex items-start justify-between gap-4">
                {/* Left: slot info */}
                <div className="flex flex-col gap-0.5">
                    <div className="flex items-center gap-2">
                        <span className="text-foreground font-medium">{slot.period_name}</span>
                        <span
                            className={`flex items-center gap-1 text-xs font-medium ${cfg.color}`}
                        >
                            {cfg.icon}
                            {cfg.label}
                        </span>
                    </div>
                    <span className="text-muted-foreground text-xs">
                        {formatTime(slot.start_time)} – {formatTime(slot.end_time)}
                    </span>
                    {slot.teacher_name && (
                        <span className="text-muted-foreground text-xs">
                            {slot.teacher_name}
                            {slot.learning_area_name && ` · ${slot.learning_area_name}`}
                        </span>
                    )}
                </div>

                {/* Right: CTA for active / missed slots */}
                {(status === "active" || status === "missed") && (
                    <Button asChild variant="outline" size="sm" className="shrink-0">
                        <Link href={`/attendance/mark/${slot.id}/${date}`}>Mark Attendance</Link>
                    </Button>
                )}

                {status === "completed" && (
                    <Badge variant="secondary" className="shrink-0 text-xs">
                        Done
                    </Badge>
                )}
            </div>
        </div>
    );
}
