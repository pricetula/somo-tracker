/**
 * TeacherAttendanceRoster — mark attendance per student per period.
 * Pure shadcn: no borders/cards, no hardcoded colours, flat layout.
 */

"use client";

import { useState, useCallback, useMemo } from "react";
import Link from "next/link";
import {
    Loader2,
    Flag,
    StickyNote,
    ClipboardList,
    AlertTriangle,
    RotateCcw,
    Lock,
} from "lucide-react";

import type { DataTableColumn } from "@/components/shared/data-table/types";
import { StaticTable } from "@/components/shared/static-table";
import { Button } from "@/components/ui/button";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Label } from "@/components/ui/label";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
    DialogTrigger,
} from "@/components/ui/dialog";
import { Textarea } from "@/components/ui/textarea";

import type { RosterStudent } from "@/lib/api/attendance";
import { AttendanceEmptyState } from "./attendance-empty-state";
import {
    useSlotRoster,
    useBulkMarkAttendance,
    useSession,
    useSkipSession,
    useUnskipSession,
} from "../hooks/use-attendance";
import type { AttendanceStatus } from "../types";
import { attendanceBadgeClass, attendanceStatusLabel } from "../types";
import { CreateBehaviorNoteDialog } from "@/features/behavior/components/create-behavior-note-dialog";

interface TeacherAttendanceRosterProps {
    timetableSlotId: string;
    date?: string;
    isLocked?: boolean;
}

export function TeacherAttendanceRoster({
    timetableSlotId,
    date,
    isLocked = false,
}: TeacherAttendanceRosterProps) {
    const { data: roster, isLoading, isError } = useSlotRoster(timetableSlotId, date);
    const {
        data: sessionData,
        isError: sessionError,
        isLoading: sessionLoading,
    } = useSession(timetableSlotId, date);
    const bulkMark = useBulkMarkAttendance();
    const skipSession = useSkipSession();
    const unskipSession = useUnskipSession();

    const [statusMap, setStatusMap] = useState<Record<string, AttendanceStatus | null>>({});
    const [noteMap, setNoteMap] = useState<Record<string, string>>({});
    const [behaviorStudent, setBehaviorStudent] = useState<RosterStudent | null>(null);
    const [skipReason, setSkipReason] = useState("");
    const [skipDialogOpen, setSkipDialogOpen] = useState(false);
    const [confirmOpen, setConfirmOpen] = useState(false);
    const [lastSavedCounts, setLastSavedCounts] = useState<Record<string, number> | null>(null);

    const isSkipped = sessionData?.session?.status === "SKIPPED";
    const skipReasonText = sessionData?.session?.skip_reason ?? null;

    const handleSkip = useCallback(() => {
        if (!skipReason.trim() || !roster) return;
        skipSession.mutate(
            {
                timetable_slot_id: roster.timetable_slot_id,
                date: roster.date,
                skip_reason: skipReason.trim(),
            },
            {
                onSettled: () => {
                    setSkipDialogOpen(false);
                    setSkipReason("");
                },
            }
        );
    }, [skipReason, roster, skipSession]);

    const handleUnskip = useCallback(() => {
        if (!roster) return;
        unskipSession.mutate({ timetable_slot_id: roster.timetable_slot_id, date: roster.date });
    }, [roster, unskipSession]);

    const effectiveStatus = useMemo(() => {
        if (!roster?.students) return {} as Record<string, AttendanceStatus | null>;
        const map: Record<string, AttendanceStatus | null> = {};
        for (const student of roster.students) {
            const localOverride = statusMap[student.student_id];
            map[student.student_id] =
                localOverride !== undefined && localOverride !== null
                    ? localOverride
                    : (student.current_status ?? null);
        }
        return map;
    }, [roster, statusMap]);

    const serverStatuses = useMemo(() => {
        if (!roster?.students) return {} as Record<string, AttendanceStatus | null>;
        const m: Record<string, AttendanceStatus | null> = {};
        for (const s of roster.students) m[s.student_id] = s.current_status ?? null;
        return m;
    }, [roster]);

    const handleSaveSuccess = useCallback(() => {
        setStatusMap({});
        setNoteMap({});
        setConfirmOpen(false);
        if (roster?.students) {
            const statusesMap = effectiveStatus;
            const counts: Record<string, number> = {};
            for (const s of roster.students) {
                const status = statusesMap[s.student_id] ?? "PRESENT";
                counts[status] = (counts[status] || 0) + 1;
            }
            setLastSavedCounts(counts);
            setTimeout(() => setLastSavedCounts(null), 8000);
        }
    }, [roster, effectiveStatus]);

    const handleMarkAllPresent = useCallback(() => {
        if (!roster?.students) return;
        const allPresent: Record<string, AttendanceStatus> = {};
        for (const student of roster.students) allPresent[student.student_id] = "PRESENT";
        setStatusMap(allPresent);
    }, [roster]);

    const handleStatusChange = useCallback((studentId: string, status: AttendanceStatus) => {
        setStatusMap((prev) => ({ ...prev, [studentId]: status }));
    }, []);

    const handleNoteChange = useCallback((studentId: string, note: string) => {
        setNoteMap((prev) => ({ ...prev, [studentId]: note }));
    }, []);

    const handleSubmitConfirmed = useCallback(() => {
        if (!roster?.students) return;
        const entries = roster.students.map((s) => ({
            student_id: s.student_id,
            status: effectiveStatus[s.student_id] ?? "PRESENT",
            note: noteMap[s.student_id] || null,
        }));
        bulkMark.mutate(
            { timetable_slot_id: roster.timetable_slot_id, date: roster.date, entries },
            { onSuccess: handleSaveSuccess }
        );
    }, [roster, effectiveStatus, noteMap, bulkMark, handleSaveSuccess]);

    const pendingSummary = useMemo(() => {
        if (!roster?.students) return null;
        const counts: Record<string, number> = {};
        let changed = 0;
        for (const s of roster.students) {
            const newStatus = effectiveStatus[s.student_id] ?? "PRESENT";
            const oldStatus = serverStatuses[s.student_id];
            counts[newStatus] = (counts[newStatus] || 0) + 1;
            if (oldStatus !== newStatus) changed++;
        }
        return { counts, changed };
    }, [roster, effectiveStatus, serverStatuses]);

    const handleSubmit = useCallback(() => {
        if (pendingSummary && pendingSummary.changed > 0) setConfirmOpen(true);
        else handleSubmitConfirmed();
    }, [pendingSummary, handleSubmitConfirmed]);

    const columns = useMemo<DataTableColumn<RosterStudent>[]>(
        () => [
            {
                id: "student",
                header: "Student",
                cell: (student) => (
                    <span
                        className={
                            isSkipped ? "text-muted-foreground" : "text-foreground font-medium"
                        }
                    >
                        {student.full_name}
                    </span>
                ),
            },
            {
                id: "status",
                header: "Status",
                cell: (student) => {
                    if (isSkipped) {
                        return (
                            <Badge variant="outline" className="text-muted-foreground">
                                Skipped
                            </Badge>
                        );
                    }
                    const status = effectiveStatus[student.student_id];
                    return isLocked ? (
                        <Badge
                            variant={status ? "outline" : "secondary"}
                            className={status ? attendanceBadgeClass(status) : ""}
                        >
                            {status ? attendanceStatusLabel(status) : "Unset"}
                        </Badge>
                    ) : (
                        <RadioGroup
                            value={effectiveStatus[student.student_id] ?? "PRESENT"}
                            onValueChange={(val) =>
                                handleStatusChange(student.student_id, val as AttendanceStatus)
                            }
                            className="flex flex-row gap-0"
                        >
                            {[
                                { value: "PRESENT", label: "Present" },
                                { value: "ABSENT", label: "Absent" },
                                { value: "LATE", label: "Late" },
                                { value: "EXCUSED", label: "Excused" },
                            ].map((option) => (
                                <div
                                    key={option.value}
                                    className="flex items-center gap-1.5 px-2 py-1"
                                >
                                    <RadioGroupItem
                                        value={option.value}
                                        id={`${student.student_id}-${option.value}`}
                                        className="size-4"
                                        disabled={isSkipped}
                                    />
                                    <Label
                                        htmlFor={`${student.student_id}-${option.value}`}
                                        className={`cursor-pointer text-xs font-normal ${isSkipped ? "text-muted-foreground" : "text-foreground"}`}
                                    >
                                        {option.label}
                                    </Label>
                                </div>
                            ))}
                        </RadioGroup>
                    );
                },
            },
            {
                id: "note",
                header: "Note",
                width: "100px",
                align: "center",
                cell: (student) =>
                    isSkipped ? null : (
                        <Popover>
                            <PopoverTrigger asChild>
                                <Button
                                    variant="ghost"
                                    size="icon"
                                    className="h-8 w-8"
                                    aria-label="Add note"
                                >
                                    <StickyNote className="text-muted-foreground h-4 w-4" />
                                </Button>
                            </PopoverTrigger>
                            <PopoverContent className="w-64" align="end">
                                <Input
                                    placeholder="Add a note..."
                                    value={noteMap[student.student_id] ?? ""}
                                    onChange={(e) =>
                                        handleNoteChange(student.student_id, e.target.value)
                                    }
                                />
                            </PopoverContent>
                        </Popover>
                    ),
            },
            {
                id: "flag",
                header: "Behaviour",
                width: "100px",
                align: "center",
                cell: (student) =>
                    isSkipped ? null : (
                        <Button
                            variant="ghost"
                            size="icon"
                            className="h-8 w-8"
                            aria-label="Log behavior note"
                            onClick={() => setBehaviorStudent(student)}
                        >
                            <Flag className="text-muted-foreground h-4 w-4" />
                        </Button>
                    ),
            },
        ],
        [isSkipped, isLocked, effectiveStatus, handleStatusChange, handleNoteChange, noteMap]
    );

    if (isLoading || sessionLoading) {
        return (
            <div className="space-y-3">
                <Skeleton className="h-8 w-64" />
                {Array.from({ length: 8 }).map((_, i) => (
                    <Skeleton key={i} className="h-10 w-full" />
                ))}
            </div>
        );
    }

    if (isError || sessionError) {
        return (
            <div className="text-destructive bg-destructive/10 p-4">
                Failed to load roster. Please try again.
            </div>
        );
    }

    if (!roster?.students?.length) {
        return (
            <AttendanceEmptyState
                icon={ClipboardList}
                title="No students enrolled"
                description="This class doesn't have any active student enrollments for the current term."
            >
                <Button variant="outline" size="sm" asChild>
                    <Link href="/students">Manage enrollments</Link>
                </Button>
            </AttendanceEmptyState>
        );
    }

    return (
        <div className="space-y-4">
            {/* Skipped alert — no border, just semantic background */}
            {isSkipped && (
                <div className="bg-muted/30 flex items-start gap-3 p-4">
                    <AlertTriangle className="text-muted-foreground mt-0.5 h-4 w-4 shrink-0" />
                    <div>
                        <p className="text-foreground font-medium">Session skipped</p>
                        <p className="text-muted-foreground text-sm">
                            This session was marked as skipped due to{" "}
                            <strong className="text-foreground">
                                {skipReasonText ?? "unspecified reason"}
                            </strong>
                            . These hours are omitted from students&apos; record cards.
                        </p>
                    </div>
                </div>
            )}

            {/* Header */}
            <div className="flex items-center justify-between">
                <div>
                    <p className="text-foreground font-semibold">{roster.class_name}</p>
                    <p className="text-muted-foreground text-sm">
                        {roster.learning_area} &middot; {roster.date}
                    </p>
                </div>
                <div className="flex items-center gap-2">
                    {!isSkipped && !isLocked && (
                        <Button variant="outline" size="sm" onClick={handleMarkAllPresent}>
                            Mark all Present
                        </Button>
                    )}

                    {isSkipped ? (
                        <Button
                            variant="outline"
                            size="sm"
                            onClick={handleUnskip}
                            disabled={unskipSession.isPending}
                        >
                            {unskipSession.isPending || skipSession.isPending ? (
                                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                            ) : (
                                <RotateCcw className="mr-2 h-4 w-4" />
                            )}
                            {skipSession.isPending ? "Skipping..." : "Undo Skip / Re-open"}
                        </Button>
                    ) : (
                        <Dialog open={skipDialogOpen} onOpenChange={setSkipDialogOpen}>
                            <DialogTrigger asChild>
                                <Button variant="outline" size="sm" disabled={isLocked}>
                                    <AlertTriangle className="mr-2 h-4 w-4" />
                                    Class Did Not Hold?
                                </Button>
                            </DialogTrigger>
                            <DialogContent>
                                <DialogHeader>
                                    <DialogTitle>Skip this session</DialogTitle>
                                    <DialogDescription>
                                        Mark this class session as skipped. All attendance records
                                        for this date will be removed, and the hours will be
                                        excluded from students&apos; terminal percentages.
                                    </DialogDescription>
                                </DialogHeader>
                                <div className="space-y-2">
                                    <Label htmlFor="skip-reason" className="text-foreground">
                                        Reason for skipping
                                    </Label>
                                    <Textarea
                                        id="skip-reason"
                                        placeholder="e.g. School Assembly, Public Holiday, Teacher Absence, Sports/Field Event"
                                        value={skipReason}
                                        onChange={(e) => setSkipReason(e.target.value)}
                                        rows={3}
                                    />
                                </div>
                                <DialogFooter>
                                    <Button
                                        variant="ghost"
                                        onClick={() => {
                                            setSkipDialogOpen(false);
                                            setSkipReason("");
                                        }}
                                    >
                                        Cancel
                                    </Button>
                                    <Button
                                        variant="destructive"
                                        onClick={handleSkip}
                                        disabled={!skipReason.trim() || skipSession.isPending}
                                    >
                                        {skipSession.isPending && (
                                            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                                        )}
                                        Skip Session
                                    </Button>
                                </DialogFooter>
                            </DialogContent>
                        </Dialog>
                    )}
                </div>
            </div>

            {/* Table */}
            <StaticTable
                columns={columns}
                data={roster.students}
                getRowId={(student) => student.student_id}
                rowHeight={48}
                height={334}
            />

            {/* Status messages — no borders, no cards */}
            {isSkipped ? (
                <p className="text-muted-foreground italic">
                    This session was skipped. Use the undo button above to re-open it.
                </p>
            ) : isLocked ? (
                <div className="bg-muted/30 flex items-start gap-2 p-4">
                    <Lock className="text-muted-foreground mt-0.5 h-4 w-4 shrink-0" />
                    <div>
                        <p className="text-foreground font-medium">
                            Past date &mdash; records locked
                        </p>
                        <p className="text-muted-foreground mt-0.5 text-xs">
                            Attendance for past dates can only be edited by a school admin. Contact
                            your admin if corrections are needed.
                        </p>
                    </div>
                </div>
            ) : (
                <p className="text-muted-foreground text-xs">
                    Same-day records can be edited until midnight. Changes are saved per class
                    session.
                </p>
            )}

            {!isSkipped && !isLocked && (
                <div className="space-y-3">
                    {lastSavedCounts && (
                        <div className="bg-muted/30 flex items-center gap-3 p-4">
                            <span className="text-emerald-600">
                                Saved:{" "}
                                {Object.entries(lastSavedCounts)
                                    .filter(([, c]) => c > 0)
                                    .map(([s, c]) => `${c} ${s.toLowerCase()}`)
                                    .join(", ")}
                            </span>
                            <Button
                                variant="outline"
                                size="sm"
                                onClick={() => {
                                    setStatusMap({});
                                    setNoteMap({});
                                    setLastSavedCounts(null);
                                }}
                                className="ml-auto h-7 text-xs"
                            >
                                <RotateCcw className="mr-1 h-3 w-3" />
                                Undo
                            </Button>
                        </div>
                    )}

                    <div className="flex items-center gap-3">
                        <Button onClick={handleSubmit} disabled={bulkMark.isPending} size="lg">
                            {bulkMark.isPending && (
                                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                            )}
                            Save Attendance
                        </Button>
                        {pendingSummary && pendingSummary.changed > 0 && (
                            <span className="text-muted-foreground text-xs">
                                {pendingSummary.changed} change
                                {pendingSummary.changed !== 1 ? "s" : ""} &middot;{" "}
                                {Object.entries(pendingSummary.counts)
                                    .filter(([, c]) => c > 0)
                                    .map(([s, c]) => `${c}\u00d7 ${s.toLowerCase()}`)
                                    .join(", ")}
                            </span>
                        )}
                    </div>
                </div>
            )}

            <Dialog open={confirmOpen} onOpenChange={setConfirmOpen}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>Confirm attendance</DialogTitle>
                        <DialogDescription>
                            You are about to save attendance for {roster?.students?.length ?? 0}{" "}
                            students.
                            {pendingSummary && pendingSummary.changed > 0 && (
                                <>
                                    {" "}
                                    {pendingSummary.changed} record
                                    {pendingSummary.changed !== 1 ? "s" : ""} will be changed.
                                </>
                            )}
                        </DialogDescription>
                    </DialogHeader>
                    {pendingSummary && (
                        <div className="space-y-2">
                            {Object.entries(pendingSummary.counts)
                                .filter(([, c]) => c > 0)
                                .map(([status, count]) => (
                                    <div key={status} className="flex items-center justify-between">
                                        <Badge
                                            variant="outline"
                                            className={attendanceBadgeClass(
                                                status as AttendanceStatus
                                            )}
                                        >
                                            {attendanceStatusLabel(status as AttendanceStatus)}
                                        </Badge>
                                        <span className="text-muted-foreground">
                                            {count} student{count !== 1 ? "s" : ""}
                                        </span>
                                    </div>
                                ))}
                        </div>
                    )}
                    <DialogFooter>
                        <Button variant="ghost" onClick={() => setConfirmOpen(false)}>
                            Cancel
                        </Button>
                        <Button onClick={handleSubmitConfirmed}>Confirm &amp; Save</Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            {behaviorStudent && (
                <CreateBehaviorNoteDialog
                    open
                    onOpenChange={(open) => {
                        if (!open) setBehaviorStudent(null);
                    }}
                    timetableSlotId={roster.timetable_slot_id}
                    studentId={behaviorStudent.student_id}
                    date={roster.date}
                />
            )}
        </div>
    );
}
