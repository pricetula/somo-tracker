"use client";

import * as React from "react";
import { format } from "date-fns";
import { CalendarDays, Coffee } from "lucide-react";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";

import { CbcTodaySlotCard } from "./cbc-today-slot-card";
import {
    useTeacherTodaySlots,
    useCbcClassStudents,
    useCbcAttendanceLogs,
    useSaveAttendanceMark,
    useMarkRemainingAsPresent,
    useCreateCbcAttendancePeriod,
} from "@/features/cbc/hooks/use-cbc-attendance";
import type {
    DashboardSlotCard as DashboardSlotCardType,
    AttendanceStatus,
    OfflineAttendanceEntry,
    AttendanceStudentRow,
} from "@/features/cbc/types";

// ─── Props ────────────────────────────────────────────────────────────────

interface MyAttendanceSectionProps {
    teacherId: string;
}

// ─── Component ────────────────────────────────────────────────────────────

export function MyAttendanceSection({ teacherId }: MyAttendanceSectionProps) {
    const today = format(new Date(), "EEEE, MMMM d, yyyy");

    // ── Slots data ───────────────────────────────────────────────────
    const { data: slots = [], isLoading, error } = useTeacherTodaySlots(teacherId);

    // ── Find the Ongoing slot ────────────────────────────────────────
    const ongoingSlot = slots.find((s) => s.status === "ongoing");

    // ── Sort slots: past_not_taken first, then by start_time ─────────
    const sortedSlots = React.useMemo(() => {
        return [...slots].sort((a, b) => {
            if (a.status === "past_not_taken" && b.status !== "past_not_taken") return -1;
            if (a.status !== "past_not_taken" && b.status === "past_not_taken") return 1;
            return a.start_time.localeCompare(b.start_time);
        });
    }, [slots]);

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
                        <Skeleton className="h-24 w-full" />
                        <Skeleton className="h-24 w-full" />
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

    // ── Empty state ──────────────────────────────────────────────────
    if (sortedSlots.length === 0) {
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
                    <div className="flex flex-col items-center justify-center rounded-md border border-dashed py-6">
                        <Coffee className="text-muted-foreground mb-2 size-6" />
                        <p className="text-muted-foreground text-xs font-medium">
                            Nothing scheduled for you today
                        </p>
                        <p className="text-muted-foreground mt-0.5 text-[10px]">
                            If you&apos;re expecting a class, check with your admin about your
                            timetable.
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
                    {sortedSlots.map((slot) => (
                        <SlotCardWithData
                            key={slot.slot_id}
                            slot={slot}
                            isPrimary={slot.slot_id === ongoingSlot?.slot_id}
                        />
                    ))}
                </div>
            </CardContent>
        </Card>
    );
}

// ─── Internal: Slot card that manages its own data and mutations ──────────

function SlotCardWithData({
    slot,
    isPrimary,
}: {
    slot: DashboardSlotCardType;
    isPrimary: boolean;
}) {
    const [expanded] = React.useState(isPrimary);
    const [localQueue, setLocalQueue] = React.useState<OfflineAttendanceEntry[]>([]);
    const [savingStudentIds, setSavingStudentIds] = React.useState<Set<string>>(new Set());

    // Only fetch data when expanded
    const { data: students = [], isLoading: studentsLoading } = useCbcClassStudents(
        expanded ? slot.class_id : null,
        expanded ? slot.academic_term_id : null
    );

    const { data: logs = [] } = useCbcAttendanceLogs(
        expanded && slot.attendance_period_id ? slot.attendance_period_id : null
    );

    const { mutateAsync: saveMark } = useSaveAttendanceMark(slot.attendance_period_id ?? "");
    const { mutateAsync: markAllPresent } = useMarkRemainingAsPresent(
        slot.attendance_period_id ?? ""
    );
    const { mutateAsync: createPeriod } = useCreateCbcAttendancePeriod(slot.class_id);

    // Merge logs into students
    const mergedStudents: AttendanceStudentRow[] = React.useMemo(() => {
        return students.map((s) => {
            const log = logs.find((l) => l.student_id === s.student_id);
            return {
                ...s,
                status: log?.status ?? null,
                log_id: log?.id ?? null,
                syncPending: false,
            };
        });
    }, [students, logs]);

    // ── Handle status selection with optimistic save ──────────────────
    const handleSelectStatus = async (
        studentId: string,
        status: AttendanceStatus,
        periodId: string
    ) => {
        const entry: OfflineAttendanceEntry = {
            localId: `${studentId}-${Date.now()}`,
            periodId,
            studentId,
            status,
            timestamp: Date.now(),
            retryCount: 0,
        };

        setLocalQueue((prev) => [...prev.filter((e) => e.studentId !== studentId), entry]);
        setSavingStudentIds((prev) => new Set(prev).add(studentId));

        try {
            await saveMark({ studentId, status });
            setLocalQueue((prev) => prev.filter((e) => e.studentId !== studentId));
        } catch {
            // Stay in queue — retry badge visible
        } finally {
            setSavingStudentIds((prev) => {
                const next = new Set(prev);
                next.delete(studentId);
                return next;
            });
        }
    };

    // ── Mark all unmarked as Present ─────────────────────────────
    const handleMarkAllPresent = async () => {
        if (!slot.attendance_period_id) return;
        const unmarkedIds = mergedStudents
            .filter((s) => s.status === null)
            .map((s) => s.student_id);
        if (unmarkedIds.length === 0) return;

        // Optimistic updates
        const entries: OfflineAttendanceEntry[] = unmarkedIds.map((id) => ({
            localId: `${id}-bulk-${Date.now()}`,
            periodId: slot.attendance_period_id!,
            studentId: id,
            status: "PRESENT" as AttendanceStatus,
            timestamp: Date.now(),
            retryCount: 0,
        }));

        setLocalQueue((prev) => [
            ...prev.filter((e) => !unmarkedIds.includes(e.studentId)),
            ...entries,
        ]);
        unmarkedIds.forEach((id) => {
            setSavingStudentIds((prev) => new Set(prev).add(id));
        });

        try {
            await markAllPresent(unmarkedIds);
            setLocalQueue((prev) => prev.filter((e) => !unmarkedIds.includes(e.studentId)));
        } catch {
            // Stay in queue
        } finally {
            unmarkedIds.forEach((id) => {
                setSavingStudentIds((prev) => {
                    const next = new Set(prev);
                    next.delete(id);
                    return next;
                });
            });
        }
    };

    // ── Start a new attendance period ────────────────────────────
    const handleStartAttendance = async () => {
        if (!slot.learning_area_id) return;
        const date = format(new Date(), "yyyy-MM-dd");
        try {
            await createPeriod({
                cbcLearningAreaId: slot.learning_area_id,
                date,
            });
        } catch {
            // handled in hook via toast
        }
    };

    return (
        <CbcTodaySlotCard
            slot={slot}
            students={mergedStudents}
            studentsLoading={studentsLoading}
            attendancePeriodId={slot.attendance_period_id}
            localQueue={localQueue}
            savingStudentIds={savingStudentIds}
            onSelectStatus={handleSelectStatus}
            onMarkAllPresent={handleMarkAllPresent}
            onStartAttendance={handleStartAttendance}
            isPrimary={isPrimary}
        />
    );
}
