"use client";

import * as React from "react";
import { format } from "date-fns";
import { CalendarDays, ChevronRight } from "lucide-react";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";

import { useTeacherTodaySlots } from "@/features/cbc/hooks/use-cbc-attendance";

// ─── Props ────────────────────────────────────────────────────────────────

interface MyAttendanceSectionProps {
    teacherId: string;
}

// ─── Component ────────────────────────────────────────────────────────────

export function MyAttendanceSection({ teacherId }: MyAttendanceSectionProps) {
    const { data: slots = [], isLoading, error } = useTeacherTodaySlots(teacherId);

    const today = format(new Date(), "EEEE, MMMM d, yyyy");

    // ── Loading state ──────────────────────────────────────────────────
    if (isLoading) {
        return (
            <Card>
                <CardHeader className="pb-2">
                    <CardTitle className="flex items-center gap-2 text-sm font-medium">
                        <CalendarDays className="size-4 text-teal-600" />
                        My attendance today
                    </CardTitle>
                </CardHeader>
                <CardContent>
                    <div className="space-y-2">
                        <Skeleton className="h-4 w-48" />
                        <Skeleton className="h-10 w-full" />
                        <Skeleton className="h-10 w-full" />
                    </div>
                </CardContent>
            </Card>
        );
    }

    // ── Error state ────────────────────────────────────────────────────
    if (error) {
        return (
            <Card>
                <CardHeader className="pb-2">
                    <CardTitle className="flex items-center gap-2 text-sm font-medium">
                        <CalendarDays className="size-4 text-teal-600" />
                        My attendance today
                    </CardTitle>
                </CardHeader>
                <CardContent>
                    <p className="text-destructive text-xs">
                        Failed to load today&apos;s attendance.
                    </p>
                </CardContent>
            </Card>
        );
    }

    // ── Empty state (no slots today) ───────────────────────────────────
    if (slots.length === 0) {
        return (
            <Card>
                <CardHeader className="pb-2">
                    <CardTitle className="flex items-center gap-2 text-sm font-medium">
                        <CalendarDays className="size-4 text-teal-600" />
                        My attendance today
                    </CardTitle>
                </CardHeader>
                <CardContent>
                    <p className="text-muted-foreground text-xs">{today}</p>
                    <div className="mt-3 flex items-center justify-center rounded-md border border-dashed py-4">
                        <p className="text-muted-foreground text-xs">
                            No classes scheduled for today
                        </p>
                    </div>
                </CardContent>
            </Card>
        );
    }

    // ── Active state ───────────────────────────────────────────────────
    return (
        <Card>
            <CardHeader className="pb-2">
                <CardTitle className="flex items-center gap-2 text-sm font-medium">
                    <CalendarDays className="size-4 text-teal-600" />
                    My attendance today
                </CardTitle>
            </CardHeader>
            <CardContent>
                <p className="text-muted-foreground mb-3 text-xs">{today}</p>

                <div className="space-y-2">
                    {slots.map((slot) => (
                        <div
                            key={slot.period_id}
                            className="flex items-center gap-3 rounded-md border px-3 py-2.5 transition-colors hover:bg-gray-50"
                        >
                            {/* Time */}
                            <div className="shrink-0 text-center">
                                <p className="text-xs font-medium">{slot.start_time}</p>
                                <p className="text-muted-foreground text-[10px]">{slot.end_time}</p>
                            </div>

                            {/* Divider */}
                            <div className="bg-border h-8 w-px" />

                            {/* Learning area */}
                            <div className="min-w-0 flex-1">
                                <p className="truncate text-sm font-medium">
                                    {slot.learning_area_name}
                                </p>
                                <p className="text-muted-foreground text-xs">
                                    {slot.start_time} – {slot.end_time}
                                </p>
                            </div>

                            {/* Status or action */}
                            <Button
                                variant="ghost"
                                size="sm"
                                className="h-8 shrink-0 text-xs"
                                asChild
                            >
                                <a href={`/classes/${slot.period_id.split("_")[0]}/attendance`}>
                                    Take attendance
                                    <ChevronRight className="ml-1 size-3" />
                                </a>
                            </Button>
                        </div>
                    ))}
                </div>
            </CardContent>
        </Card>
    );
}
