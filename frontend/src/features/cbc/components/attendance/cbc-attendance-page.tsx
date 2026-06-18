"use client";

import * as React from "react";
import { format } from "date-fns";
import { ChevronLeft, ChevronRight } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import {
    Select,
    SelectTrigger,
    SelectValue,
    SelectContent,
    SelectItem,
} from "@/components/ui/select";

import { CbcAttendanceGrid } from "./cbc-attendance-grid";
import {
    useCbcAttendancePeriods,
    useCreateCbcAttendancePeriod,
    useCbcAttendanceLogs,
    useCbcClassStudents,
    useSaveAttendanceMark,
} from "@/features/cbc/hooks/use-cbc-attendance";
import { useCbcLearningAreas } from "@/features/cbc/hooks/use-cbc-timetable";
import type {
    AttendanceStudentRow,
    AttendanceStatus,
    OfflineAttendanceEntry,
} from "@/features/cbc/types";

// ─── Props ────────────────────────────────────────────────────────────────

interface CbcAttendancePageProps {
    classId: string;
    schoolId: string;
    academicTermId: string;
    gradeId: string;
    className: string;
}

// ─── Component ────────────────────────────────────────────────────────────

export function CbcAttendancePage({
    classId,
    academicTermId,
    gradeId,
    className,
}: CbcAttendancePageProps) {
    // ── Date state ─────────────────────────────────────────────────────
    const [selectedDate, setSelectedDate] = React.useState(format(new Date(), "yyyy-MM-dd"));

    // ── Learning area state ───────────────────────────────────────────
    const [selectedLearningArea, setSelectedLearningArea] = React.useState<string>("");

    // ── Offline optimistic queue ──────────────────────────────────────
    const [localQueue, setLocalQueue] = React.useState<OfflineAttendanceEntry[]>([]);
    const [savingStudentIds, setSavingStudentIds] = React.useState<Set<string>>(new Set());

    // ── Data ────────────────────────────────────────────────────────────
    const { data: learningAreas = [] } = useCbcLearningAreas(gradeId);

    const { data: periods = [], isLoading: periodsLoading } = useCbcAttendancePeriods(
        classId,
        selectedDate
    );

    const { data: students = [], isLoading: studentsLoading } = useCbcClassStudents(
        classId,
        academicTermId
    );

    // Find the attendance period matching the selected learning area
    const matchingPeriod = periods.find((p) => p.cbc_learning_area_id === selectedLearningArea);

    const { data: logs = [] } = useCbcAttendanceLogs(matchingPeriod?.id ?? null);

    const { mutateAsync: createPeriod } = useCreateCbcAttendancePeriod(classId);
    const { mutateAsync: saveMark } = useSaveAttendanceMark(matchingPeriod?.id ?? "");

    // ── Merge logs into student rows ──────────────────────────────────
    const studentRows: AttendanceStudentRow[] = React.useMemo(() => {
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

    // ── Handle status selection ───────────────────────────────────────
    const handleSelectStatus = async (
        studentId: string,
        status: AttendanceStatus,
        periodId: string
    ) => {
        // Add to local optimistic queue
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
            // Remove from local queue on success
            setLocalQueue((prev) => prev.filter((e) => e.studentId !== studentId));
        } catch {
            // Mark stays in queue for retry; retry badge visible
        } finally {
            setSavingStudentIds((prev) => {
                const next = new Set(prev);
                next.delete(studentId);
                return next;
            });
        }
    };

    // ── Start a new attendance period for the selected learning area ──
    const handleStartAttendance = async () => {
        if (!selectedLearningArea) return;
        try {
            await createPeriod({
                cbcLearningAreaId: selectedLearningArea,
                date: selectedDate,
            });
        } catch {
            // Error handled in the hook (toast)
        }
    };

    // ── Navigate date ─────────────────────────────────────────────────
    const goToPreviousDay = () => {
        const d = new Date(selectedDate);
        d.setDate(d.getDate() - 1);
        setSelectedDate(format(d, "yyyy-MM-dd"));
    };

    const goToNextDay = () => {
        const d = new Date(selectedDate);
        d.setDate(d.getDate() + 1);
        // Don't allow future dates beyond today
        if (d <= new Date()) {
            setSelectedDate(format(d, "yyyy-MM-dd"));
        }
    };

    const goToToday = () => {
        setSelectedDate(today);
    };

    // ── Date display ──────────────────────────────────────────────────
    const displayDate = format(new Date(selectedDate), "EEE, MMM d, yyyy");

    // ── Render ────────────────────────────────────────────────────────
    return (
        <div className="flex flex-1 flex-col gap-4">
            {/* ── Header ─────────────────────────────────────────────── */}
            <div className="flex items-center justify-between">
                <h2 className="text-lg font-medium tracking-tight">{className} — attendance</h2>
            </div>

            {/* ── Date navigation ────────────────────────────────────── */}
            <div className="flex items-center gap-2">
                <div className="flex items-center gap-1">
                    <Button
                        variant="ghost"
                        size="icon"
                        className="size-8"
                        onClick={goToPreviousDay}
                        aria-label="Previous day"
                    >
                        <ChevronLeft className="size-4" />
                    </Button>

                    <span className="min-w-40 text-center text-sm font-medium">{displayDate}</span>

                    <Button
                        variant="ghost"
                        size="icon"
                        className="size-8"
                        onClick={goToNextDay}
                        disabled={format(new Date(selectedDate), "yyyy-MM-dd") >= today}
                        aria-label="Next day"
                    >
                        <ChevronRight className="size-4" />
                    </Button>
                </div>

                {selectedDate !== today && (
                    <Button variant="outline" size="sm" className="h-7 text-xs" onClick={goToToday}>
                        Today
                    </Button>
                )}
            </div>

            {/* ── Learning area selector ─────────────────────────────── */}
            <div className="flex items-center gap-3">
                <div className="w-64">
                    <Select value={selectedLearningArea} onValueChange={setSelectedLearningArea}>
                        <SelectTrigger>
                            <SelectValue placeholder="Select learning area..." />
                        </SelectTrigger>
                        <SelectContent>
                            {learningAreas.map((la) => (
                                <SelectItem key={la.id} value={la.id}>
                                    {la.name}
                                </SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                </div>

                {selectedLearningArea && !matchingPeriod && (
                    <Button
                        variant="outline"
                        size="sm"
                        onClick={handleStartAttendance}
                        className="h-8 text-xs"
                    >
                        Start attendance for this period
                    </Button>
                )}

                {matchingPeriod && (
                    <span className="flex items-center gap-1 text-xs font-medium text-teal-600">
                        <span className="inline-block size-1.5 rounded-full bg-teal-500" />
                        Attendance in progress
                    </span>
                )}
            </div>

            {/* ── Attendance grid ────────────────────────────────────── */}
            <Card className="overflow-hidden">
                {periodsLoading || studentsLoading ? (
                    <div className="p-4">
                        <Skeleton className="h-8 w-full" />
                        <div className="mt-2 space-y-2">
                            {Array.from({ length: 5 }).map((_, i) => (
                                <Skeleton key={i} className="h-12 w-full" />
                            ))}
                        </div>
                    </div>
                ) : (
                    <CbcAttendanceGrid
                        students={studentRows}
                        isLoading={studentsLoading}
                        periodId={matchingPeriod?.id ?? null}
                        localQueue={localQueue}
                        onSelectStatus={handleSelectStatus}
                        savingStudentIds={savingStudentIds}
                    />
                )}
            </Card>
        </div>
    );
}
