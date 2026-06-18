"use client";

import * as React from "react";
import { format } from "date-fns";
import { ArrowLeft, ClipboardList } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { CbcAttendanceHeatmap } from "./cbc-attendance-heatmap";
import { CbcAttendancePeriodList } from "./cbc-attendance-period-list";
import { CbcAttendanceGrid } from "./cbc-attendance-grid";
import {
    useCbcAttendancePeriods,
    useCbcAttendancePeriodSummaries,
    useCbcAttendanceLogs,
    useCbcClassStudents,
    useSaveAttendanceMark,
    useMarkRemainingAsPresent,
    useCreateCbcAttendancePeriod,
    useCbcAttendanceHeatmap,
} from "@/features/cbc/hooks/use-cbc-attendance";
import { useCbcLearningAreas } from "@/features/cbc/hooks/use-cbc-timetable";
import type {
    AttendanceStudentRow,
    AttendanceStatus,
    AttendanceGap,
    OfflineAttendanceEntry,
} from "@/features/cbc/types";

// ─── Props ────────────────────────────────────────────────────────────────

interface CbcAttendancePageProps {
    classId: string;
    schoolId: string;
    academicTermId: string;
    gradeId: string;
    className: string;
    /** Current user's ID for permission checking. */
    userId?: string;
    /** Current user's role. */
    userRole?: "SYSTEM_ADMIN" | "SCHOOL_ADMIN" | "TEACHER" | "SUPPORT_STAFF";
}

// ─── Component ────────────────────────────────────────────────────────────

export function CbcAttendancePage({
    classId,
    academicTermId,
    gradeId,
    className,
    userRole = "TEACHER",
}: CbcAttendancePageProps) {
    // ── View state ───────────────────────────────────────────────────
    const [view, setView] = React.useState<"list" | "register">("list");
    const [selectedPeriodId, setSelectedPeriodId] = React.useState<string | null>(null);
    const [selectedDate, setSelectedDate] = React.useState(format(new Date(), "yyyy-MM-dd"));

    // ── Offline optimistic queue ──────────────────────────────────────
    const [localQueue, setLocalQueue] = React.useState<OfflineAttendanceEntry[]>([]);
    const [savingStudentIds, setSavingStudentIds] = React.useState<Set<string>>(new Set());

    // ── Local remarks store (studentId → remarks text) ───────────────
    const [remarksStore, setRemarksStore] = React.useState<Record<string, string>>({});

    // ── Data ────────────────────────────────────────────────────────────
    const { data: learningAreas = [] } = useCbcLearningAreas(gradeId);
    const { data: heatmapData = [], isLoading: heatmapLoading } = useCbcAttendanceHeatmap(
        classId,
        academicTermId
    );

    const { data: periods = [], isLoading: periodsLoading } = useCbcAttendancePeriods(
        classId,
        selectedDate
    );

    // Monthly summaries used for heatmap legend / stats
    useCbcAttendancePeriodSummaries(
        classId,
        format(new Date(new Date().getFullYear(), new Date().getMonth(), 1), "yyyy-MM-dd"),
        format(new Date(), "yyyy-MM-dd")
    );

    const { data: students = [], isLoading: studentsLoading } = useCbcClassStudents(
        classId,
        academicTermId
    );

    const { data: logs = [] } = useCbcAttendanceLogs(selectedPeriodId);

    // Selected period details
    const selectedPeriod = periods.find((p) => p.id === selectedPeriodId);

    const { mutateAsync: createPeriod } = useCreateCbcAttendancePeriod(classId);
    const { mutateAsync: saveMark } = useSaveAttendanceMark(selectedPeriodId ?? "");
    const { mutateAsync: markRemaining } = useMarkRemainingAsPresent(selectedPeriodId ?? "");

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
        // Add to optimistic queue
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
            await saveMark({ studentId, status, remarks: remarksStore[studentId] ?? undefined });
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

    // ── Handle remarks change ─────────────────────────────────────────
    const handleRemarksChange = (studentId: string, remarks: string) => {
        setRemarksStore((prev) => ({ ...prev, [studentId]: remarks }));
    };

    // ── Handle Mark Remaining as Present ──────────────────────────────
    const handleMarkRemaining = async (studentIds: string[]) => {
        if (!selectedPeriodId || studentIds.length === 0) return;

        // Optimistically update local queue
        const entries: OfflineAttendanceEntry[] = studentIds.map((id) => ({
            localId: `${id}-bulk-${Date.now()}`,
            periodId: selectedPeriodId,
            studentId: id,
            status: "PRESENT" as AttendanceStatus,
            timestamp: Date.now(),
            retryCount: 0,
        }));

        setLocalQueue((prev) => [
            ...prev.filter((e) => !studentIds.includes(e.studentId)),
            ...entries,
        ]);
        studentIds.forEach((id) => {
            setSavingStudentIds((prev) => new Set(prev).add(id));
        });

        try {
            await markRemaining(studentIds);
            setLocalQueue((prev) => prev.filter((e) => !studentIds.includes(e.studentId)));
        } catch {
            // Stay in queue
        } finally {
            studentIds.forEach((id) => {
                setSavingStudentIds((prev) => {
                    const next = new Set(prev);
                    next.delete(id);
                    return next;
                });
            });
        }
    };

    // ── Handle selecting a period from the list ───────────────────────
    const handleSelectPeriod = (periodId: string) => {
        setSelectedPeriodId(periodId);
        setView("register");
    };

    // ── Handle filling a gap ─────────────────────────────────────────
    const handleFillGap = async (gap: AttendanceGap) => {
        if (!gap.cbc_learning_area_id) return;

        try {
            const period = await createPeriod({
                cbcLearningAreaId: gap.cbc_learning_area_id,
                date: gap.date,
            });
            setSelectedPeriodId(period.id);
            setSelectedDate(gap.date);
            setView("register");
        } catch {
            // Error toast handled by hook
        }
    };

    // ── Handle starting a new period for a learning area on a date ────
    const handleStartNewPeriod = async (learningAreaId: string, date: string) => {
        try {
            const period = await createPeriod({
                cbcLearningAreaId: learningAreaId,
                date,
            });
            setSelectedPeriodId(period.id);
            setSelectedDate(date);
            setView("register");
        } catch {
            // Error toast handled by hook
        }
    };

    // ── Go back to list view ─────────────────────────────────────────
    const handleBackToList = () => {
        setSelectedPeriodId(null);
        setView("list");
    };

    // ── Check permissions ────────────────────────────────────────────
    // SCHOOL_ADMIN can edit all periods; TEACHER can edit if they
    // are the recorded_by for the period. The backend enforces the
    // canonical check against cbc_class_teachers.
    const isNewPeriod = selectedPeriodId !== null && !selectedPeriod;
    const canEdit =
        userRole === "SCHOOL_ADMIN" ||
        (userRole === "TEACHER" && (isNewPeriod || selectedPeriod?.id === selectedPeriodId));

    const learningAreaOptions = learningAreas.map((la) => ({
        id: la.id,
        name: la.name,
    }));

    // ── Render ────────────────────────────────────────────────────────
    return (
        <div className="flex flex-1 flex-col gap-4">
            {/* ── Header ─────────────────────────────────────────────── */}
            <div className="flex items-center justify-between">
                <h2 className="text-lg font-medium tracking-tight">{className} — attendance</h2>
            </div>

            {/* ── Heatmap section ────────────────────────────────────── */}
            <Card className="overflow-hidden p-4">
                {heatmapLoading ? (
                    <div className="space-y-2">
                        <Skeleton className="h-4 w-32" />
                        <div className="grid grid-cols-7 gap-1">
                            {Array.from({ length: 35 }).map((_, i) => (
                                <Skeleton key={i} className="size-8 rounded" />
                            ))}
                        </div>
                    </div>
                ) : (
                    <CbcAttendanceHeatmap
                        data={heatmapData}
                        selectedDate={selectedDate}
                        onSelectDate={(date) => {
                            setSelectedDate(date);
                            if (view === "register") {
                                setView("list");
                                setSelectedPeriodId(null);
                            }
                        }}
                        termStart={format(new Date(new Date().getFullYear(), 0, 1), "yyyy-MM-dd")}
                        termEnd={format(new Date(), "yyyy-MM-dd")}
                    />
                )}
            </Card>

            {/* ── Main content: List or Register ─────────────────────── */}
            {view === "list" && (
                <Card className="overflow-hidden">
                    <div className="border-b px-3 py-2">
                        <div className="flex items-center gap-2">
                            <ClipboardList className="size-4 text-teal-600" />
                            <span className="text-sm font-medium">Attendance records</span>
                        </div>
                    </div>
                    <CbcAttendancePeriodList
                        classId={classId}
                        learningAreaOptions={learningAreaOptions}
                        selectedPeriodId={selectedPeriodId}
                        onSelectPeriod={handleSelectPeriod}
                        onFillGap={handleFillGap}
                        onStartNewPeriod={handleStartNewPeriod}
                        academicTermId={academicTermId}
                    />
                </Card>
            )}

            {view === "register" && (
                <div className="space-y-3">
                    {/* Back button */}
                    <Button
                        variant="ghost"
                        size="sm"
                        className="h-7 px-2 text-xs"
                        onClick={handleBackToList}
                    >
                        <ArrowLeft className="mr-1 size-3.5" />
                        Back to records
                    </Button>

                    {/* Register */}
                    <Card className="overflow-hidden">
                        <div className="border-b bg-gray-50 px-3 py-2">
                            <p className="text-xs font-medium">
                                {selectedPeriod
                                    ? `${format(new Date(selectedPeriod.date_recorded), "MMM d, yyyy")} — ${learningAreas.find((la) => la.id === selectedPeriod.cbc_learning_area_id)?.name ?? "Unknown"}`
                                    : "Attendance register"}
                            </p>
                        </div>
                        <CbcAttendanceGrid
                            students={studentRows}
                            logs={logs}
                            isLoading={studentsLoading || periodsLoading}
                            periodId={selectedPeriodId}
                            localQueue={localQueue}
                            onSelectStatus={handleSelectStatus}
                            onRemarksChange={handleRemarksChange}
                            onMarkRemainingAsPresent={handleMarkRemaining}
                            savingStudentIds={savingStudentIds}
                            canEdit={canEdit}
                            recordedByUserId={selectedPeriod?.id}
                        />
                    </Card>
                </div>
            )}
        </div>
    );
}
