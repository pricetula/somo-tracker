/**
 * Editable attendance marking form for a timetable allocation + date.
 *
 * Reads via GET /api/v1/attendance/marked-timetable-allocation/:id
 * Saves via POST /api/v1/attendance/records/batch (bulk attendance creation).
 */
"use client";

import React, { useState, useCallback } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { format, parseISO } from "date-fns";
import {
    getMarkedTimetableAllocation,
    batchMarkAttendance,
    type BatchMarkPayload,
    type StudentMarkPayload,
    type StudentMarkingRecord,
} from "@/lib/api/attendance";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { getErrorMessage } from "@/lib/errors";

const ATTENDANCE_STATUSES = [
    { value: "PRESENT", label: "Present", color: "text-green-600" },
    { value: "ABSENT", label: "Absent", color: "text-red-600" },
    { value: "LATE", label: "Late", color: "text-yellow-600" },
    { value: "EXCUSED", label: "Excused", color: "text-blue-600" },
] as const;

type AttendanceStatus = (typeof ATTENDANCE_STATUSES)[number]["value"];

interface Props {
    allocationId: string;
    date: string;
}

interface DraftRecord {
    status: AttendanceStatus | "";
    note: string;
}

const EMPTY_DRAFT: DraftRecord = { status: "", note: "" };

function buildInitialDraft(students: StudentMarkingRecord[]): Record<string, DraftRecord> {
    const initial: Record<string, DraftRecord> = {};
    for (const student of students) {
        initial[student.student_id] = {
            status: (student.status || "") as AttendanceStatus | "",
            note: student.note ?? "",
        };
    }
    return initial;
}

export function MarkedAllocationView({ allocationId, date }: Props) {
    const qc = useQueryClient();

    const [edits, setEdits] = useState<Record<string, Partial<DraftRecord>>>({});

    const { data, isLoading, isError, error } = useQuery({
        queryKey: ["marked-allocation", allocationId, date],
        queryFn: () => getMarkedTimetableAllocation(allocationId, date),
        enabled: !!allocationId && !!date,
    });

    // Derive the live draft from server base + user edits (no effect needed).
    const draft: Record<string, DraftRecord> = React.useMemo(() => {
        const base = data?.students ? buildInitialDraft(data.students) : {};
        const merged: Record<string, DraftRecord> = {};
        for (const id of Object.keys(base)) {
            merged[id] = { ...base[id], ...(edits[id] ?? {}) };
        }
        // Preserve edits for students that may not exist in base (shouldn't happen)
        for (const id of Object.keys(edits)) {
            if (!merged[id]) merged[id] = { ...EMPTY_DRAFT, ...(edits[id] ?? {}) };
        }
        return merged;
    }, [data, edits]);

    const saveMutation = useMutation({
        mutationFn: (payload: BatchMarkPayload) => batchMarkAttendance(payload),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: ["marked-allocation", allocationId, date] });
            setEdits({});
        },
    });

    const handleStatusChange = useCallback((studentId: string, status: AttendanceStatus | "") => {
        setEdits((prev) => ({
            ...prev,
            [studentId]: { ...(prev[studentId] ?? {}), status },
        }));
    }, []);

    const handleNoteChange = useCallback((studentId: string, note: string) => {
        setEdits((prev) => ({
            ...prev,
            [studentId]: { ...(prev[studentId] ?? {}), note },
        }));
    }, []);

    const handleSubmit = useCallback(() => {
        if (!data) return;

        const records: StudentMarkPayload[] = data.students
            .map((s) => {
                const d = draft[s.student_id];
                if (!d?.status) return null;
                return {
                    student_id: s.student_id,
                    status: d.status,
                    note: d.note.trim() ? d.note.trim() : null,
                };
            })
            .filter((r): r is NonNullable<typeof r> => r !== null);

        if (records.length === 0) return;

        saveMutation.mutate({
            date,
            timetable_allocation_id: allocationId,
            records,
        });
    }, [data, draft, date, allocationId, saveMutation]);

    if (isLoading) {
        return (
            <div className="space-y-4 p-6">
                <Skeleton className="h-6 w-48" />
                <Skeleton className="h-20 w-full" />
                <Skeleton className="h-4 w-32" />
                <div className="space-y-3">
                    {Array.from({ length: 4 }).map((_, i) => (
                        <Skeleton key={i} className="h-24 w-full" />
                    ))}
                </div>
            </div>
        );
    }

    if (isError) {
        return (
            <div className="p-6">
                <Alert variant="destructive">
                    <AlertDescription>
                        {getErrorMessage(error) || "Failed to load attendance data."}
                    </AlertDescription>
                </Alert>
            </div>
        );
    }

    if (!data) {
        return (
            <div className="text-muted-foreground p-6 text-sm">No attendance data available.</div>
        );
    }

    const students = data.students;
    const sessionLocked = data.session_status === "SKIPPED";
    const markedCount = students.filter((s) => draft[s.student_id]?.status).length;

    let headerDate = date;
    try {
        headerDate = format(parseISO(date), "EEEE, MMMM d, yyyy");
    } catch {
        // fallback to raw date string
    }

    return (
        <div className="flex flex-1 flex-col gap-4 overflow-hidden">
            {/* Header */}
            <div className="border-b pb-4">
                <div className="flex flex-wrap items-baseline justify-between gap-2">
                    <div>
                        <h2 className="text-foreground text-base font-semibold">
                            {data.class_name}
                        </h2>
                        <p className="text-muted-foreground text-xs">
                            <span className="font-medium">{data.subject_name}</span>
                            {" · "}
                            <span>Teacher: {data.teacher_name}</span>
                            {data.room_identifier ? ` · Room: ${data.room_identifier}` : ""}
                        </p>
                    </div>
                    <div className="text-right">
                        <p className="text-foreground text-sm font-medium">{headerDate}</p>
                        <p className="text-muted-foreground text-xs">{date}</p>
                    </div>
                </div>

                {sessionLocked && (
                    <p className="text-muted-foreground mt-2 text-xs italic">
                        Session skipped{data.skip_reason ? ` — ${data.skip_reason}` : ""}.
                        Attendance is locked.
                    </p>
                )}
            </div>

            {/* Bulk progress */}
            <div className="flex items-center justify-between">
                <p className="text-muted-foreground text-xs">
                    {markedCount} / {students.length} marked
                </p>
            </div>

            {/* Student roster */}
            <div className="flex-1 space-y-3 overflow-y-auto pr-1">
                {students.length === 0 && (
                    <p className="text-muted-foreground py-4 text-center text-sm">
                        No students enrolled in this class for the active term.
                    </p>
                )}
                {students.map((student) => (
                    <StudentRow
                        key={student.student_id}
                        studentId={student.student_id}
                        studentName={student.student_name}
                        draft={draft[student.student_id] ?? EMPTY_DRAFT}
                        disabled={sessionLocked}
                        onStatusChange={handleStatusChange}
                        onNoteChange={handleNoteChange}
                    />
                ))}
            </div>

            {/* Footer */}
            <div className="flex items-center justify-end gap-2 border-t pt-4">
                {saveMutation.isError && (
                    <p className="text-destructive text-xs">
                        {getErrorMessage(saveMutation.error)}
                    </p>
                )}
                {saveMutation.isSuccess && <p className="text-muted-foreground text-xs">Saved.</p>}
                <Button
                    onClick={handleSubmit}
                    disabled={sessionLocked || saveMutation.isPending || markedCount === 0}
                >
                    {saveMutation.isPending ? "Saving…" : "Save attendance"}
                </Button>
            </div>
        </div>
    );
}

// ─── StudentRow ──────────────────────────────────────────────────────────

interface StudentRowProps {
    studentId: string;
    studentName: string;
    draft: DraftRecord;
    disabled: boolean;
    onStatusChange: (studentId: string, status: AttendanceStatus | "") => void;
    onNoteChange: (studentId: string, note: string) => void;
}

function StudentRow({
    studentId,
    studentName,
    draft,
    disabled,
    onStatusChange,
    onNoteChange,
}: StudentRowProps) {
    return (
        <div className="bg-card/40 space-y-3 rounded-md border p-3">
            <Label className="text-foreground text-sm font-medium">{studentName}</Label>

            {/* Status checkboxes — one per attendance status */}
            <div className="flex flex-wrap gap-x-4 gap-y-1.5">
                {ATTENDANCE_STATUSES.map((s) => (
                    <div key={s.value} className="flex items-center gap-1.5">
                        <Checkbox
                            id={`${studentId}-${s.value}`}
                            checked={draft.status === s.value}
                            disabled={disabled}
                            onCheckedChange={() => {
                                if (draft.status === s.value) {
                                    onStatusChange(studentId, "");
                                } else {
                                    onStatusChange(studentId, s.value);
                                }
                            }}
                        />
                        <Label
                            htmlFor={`${studentId}-${s.value}`}
                            className={`cursor-pointer text-xs font-medium select-none ${s.color}`}
                        >
                            {s.label}
                        </Label>
                    </div>
                ))}
            </div>

            {/* Optional note */}
            <Input
                value={draft.note}
                onChange={(e) => onNoteChange(studentId, e.target.value)}
                placeholder="Note (optional)"
                disabled={disabled}
                className="h-8 text-xs"
            />
        </div>
    );
}
