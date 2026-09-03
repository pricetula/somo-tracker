/**
 * AttendanceMarkingForm — student roll-call grid for a single slot+date.
 *
 * Behaviour:
 *   • Fetches slot details, session, and student records separately
 *   • Pre-fills checkboxes from existing records (blank when unmarked)
 *   • On submit, POSTs a BatchMarkPayload to the backend
 *   • Re-fetches records on success
 *
 * The grid renders one row per student: name, 4 status radio-style buttons
 * (Present / Absent / Late / Excused), and an optional note field.
 */
"use client";

import React, { useMemo, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { format, parseISO } from "date-fns";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";
import { getErrorMessage } from "@/lib/errors";

import {
    useSlotDetails,
    useSlotSession,
    useSlotRecords,
    attendanceMarkingKeys,
} from "../hooks/use-attendance-marking";
import {
    batchMarkAttendance,
    type StudentAttendanceRecord,
    type StudentMarkPayload,
} from "@/lib/api/attendance";
import { ATTENDANCE_STATUSES, type AttendanceStatus } from "../types";

interface AttendanceMarkingFormProps {
    allocationId: string;
    date: string; // YYYY-MM-DD
}

interface DraftRecord {
    status: AttendanceStatus | "";
    note: string;
}

export function AttendanceMarkingForm({ allocationId, date }: AttendanceMarkingFormProps) {
    const slotQ = useSlotDetails(allocationId);
    const sessionQ = useSlotSession(allocationId, date);
    const recordsQ = useSlotRecords(allocationId, date);
    const qc = useQueryClient();

    const [draft, setDraft] = useState<Record<string, DraftRecord>>({});

    // Build the initial draft from server records (once)
    const initialized = React.useRef(false);
    React.useEffect(() => {
        if (recordsQ.data && !initialized.current) {
            initialized.current = true;
            const initial: Record<string, DraftRecord> = {};
            for (const r of recordsQ.data.items) {
                initial[r.student_id] = {
                    status: (r.status || "") as AttendanceStatus | "",
                    note: r.note ?? "",
                };
            }
            setDraft(initial);
        }
    }, [recordsQ.data]);

    const isLoading = slotQ.isLoading || sessionQ.isLoading || recordsQ.isLoading;
    const isError = slotQ.isError || sessionQ.isError || recordsQ.isError;
    const error = slotQ.error ?? sessionQ.error ?? recordsQ.error;

    const submitMutation = useMutation({
        mutationFn: (records: StudentMarkPayload[]) =>
            batchMarkAttendance({
                date,
                timetable_allocation_id: allocationId,
                records,
            }),
        onSuccess: () => {
            qc.invalidateQueries({
                queryKey: attendanceMarkingKeys.records(allocationId, date),
            });
            qc.invalidateQueries({
                queryKey: attendanceMarkingKeys.session(allocationId, date),
            });
        },
    });

    const students = recordsQ.data?.items ?? [];
    const session = sessionQ.data;
    const slot = slotQ.data;

    const headerDate = useMemo(() => {
        try {
            return format(parseISO(date), "EEEE, MMMM d, yyyy");
        } catch {
            return date;
        }
    }, [date]);

    const sessionLocked = session?.status === "SKIPPED";

    const handleStatusChange = (studentId: string, status: AttendanceStatus) => {
        setDraft((d) => ({
            ...d,
            [studentId]: { ...(d[studentId] ?? { note: "" }), status },
        }));
    };

    const handleNoteChange = (studentId: string, note: string) => {
        setDraft((d) => ({
            ...d,
            [studentId]: { ...(d[studentId] ?? { status: "" }), note },
        }));
    };

    const handleMarkAll = (status: AttendanceStatus) => {
        const next: Record<string, DraftRecord> = { ...draft };
        for (const s of students) {
            next[s.student_id] = { ...(next[s.student_id] ?? { note: "" }), status };
        }
        setDraft(next);
    };

    const handleSubmit = () => {
        const records: StudentMarkPayload[] = students
            .map((s) => {
                const d = draft[s.student_id];
                if (!d?.status) return null;
                return {
                    student_id: s.student_id,
                    status: d.status as Exclude<AttendanceStatus, "">,
                    note: d.note?.trim() ? d.note.trim() : null,
                };
            })
            .filter((r): r is NonNullable<typeof r> => r !== null);

        if (records.length === 0) return;
        submitMutation.mutate(records);
    };

    if (isLoading) {
        return <div className="text-muted-foreground p-6 text-sm">Loading attendance…</div>;
    }
    if (isError) {
        return (
            <div className="bg-background rounded-md p-6">
                <p className="text-destructive text-sm font-medium">
                    {error ? getErrorMessage(error) : "Failed to load attendance."}
                </p>
            </div>
        );
    }

    if (students.length === 0) {
        return (
            <div className="bg-background flex items-center justify-center p-8">
                <p className="text-muted-foreground text-sm">
                    No students are enrolled in this class.
                </p>
            </div>
        );
    }

    const markedCount = students.filter((s) => draft[s.student_id]?.status).length;

    return (
        <div className="flex flex-1 flex-col gap-4 overflow-hidden">
            {/* ─── Header (slot details) ──────────────────────────────── */}
            <div className="border-b pb-4">
                <div className="flex flex-wrap items-baseline justify-between gap-2">
                    <div>
                        <h2 className="text-foreground text-base font-semibold">
                            {slot?.learning_area_name ?? "Subject"}
                        </h2>
                        <p className="text-muted-foreground text-xs">
                            {slot?.class_name}
                            {slot?.room_identifier ? ` · ${slot.room_identifier}` : ""}
                        </p>
                    </div>
                    <div className="text-right">
                        <p className="text-foreground text-sm font-medium">{headerDate}</p>
                        <p className="text-muted-foreground text-xs">
                            {students[0]?.period_name ?? "Period"}
                        </p>
                    </div>
                </div>
                {session?.status === "SKIPPED" && (
                    <p className="text-muted-foreground mt-2 text-xs italic">
                        Session skipped{session.skip_reason ? ` — ${session.skip_reason}` : ""}.
                    </p>
                )}
            </div>

            {/* ─── Bulk actions ────────────────────────────────────────── */}
            <div className="flex flex-wrap items-center justify-between gap-2">
                <div className="text-muted-foreground text-xs">
                    {markedCount} / {students.length} marked
                </div>
                <div className="flex gap-1">
                    {ATTENDANCE_STATUSES.map((s) => (
                        <Button
                            key={s.value}
                            type="button"
                            variant="ghost"
                            size="sm"
                            disabled={sessionLocked}
                            onClick={() => handleMarkAll(s.value)}
                            className="h-7 px-2 text-xs"
                        >
                            Mark all {s.label}
                        </Button>
                    ))}
                </div>
            </div>

            {/* ─── Student grid ────────────────────────────────────────── */}
            <div className="flex-1 space-y-2 overflow-y-auto pr-1">
                {students.map((s, idx) => (
                    <StudentRow
                        key={s.student_id}
                        index={idx + 1}
                        student={s}
                        draft={draft[s.student_id] ?? { status: "", note: "" }}
                        disabled={sessionLocked}
                        onStatusChange={(status) => handleStatusChange(s.student_id, status)}
                        onNoteChange={(note) => handleNoteChange(s.student_id, note)}
                    />
                ))}
            </div>

            {/* ─── Footer (submit) ─────────────────────────────────────── */}
            <div className="flex items-center justify-end gap-2 border-t pt-4">
                {submitMutation.isError && (
                    <p className="text-destructive text-xs">
                        {getErrorMessage(submitMutation.error)}
                    </p>
                )}
                {submitMutation.isSuccess && (
                    <p className="text-muted-foreground text-xs">Saved.</p>
                )}
                <Button
                    type="button"
                    onClick={handleSubmit}
                    disabled={sessionLocked || submitMutation.isPending || markedCount === 0}
                >
                    {submitMutation.isPending ? "Saving…" : "Save attendance"}
                </Button>
            </div>
        </div>
    );
}

// ─── StudentRow ──────────────────────────────────────────────────────────

interface StudentRowProps {
    index: number;
    student: StudentAttendanceRecord;
    draft: DraftRecord;
    disabled: boolean;
    onStatusChange: (status: AttendanceStatus) => void;
    onNoteChange: (note: string) => void;
}

function StudentRow({
    index,
    student,
    draft,
    disabled,
    onStatusChange,
    onNoteChange,
}: StudentRowProps) {
    return (
        <div className="bg-card/40 space-y-2 rounded-md border p-3">
            <div className="flex items-center justify-between gap-2">
                <Label className="text-foreground text-sm font-medium">
                    <span className="text-muted-foreground mr-2 text-xs font-normal">{index}.</span>
                    {student.student_full_name}
                </Label>
            </div>
            <div className="grid grid-cols-2 gap-1 sm:grid-cols-4">
                {ATTENDANCE_STATUSES.map((s) => {
                    const active = draft.status === s.value;
                    return (
                        <button
                            key={s.value}
                            type="button"
                            disabled={disabled}
                            onClick={() => onStatusChange(s.value)}
                            className={cn(
                                "border-input bg-background hover:bg-accent/40 flex items-center justify-center gap-1.5 rounded-md border px-2 py-1.5 text-xs font-medium transition",
                                active && s.activeClass
                            )}
                        >
                            {s.icon}
                            {s.label}
                        </button>
                    );
                })}
            </div>
            <Input
                value={draft.note}
                onChange={(e) => onNoteChange(e.target.value)}
                placeholder="Note (optional)"
                disabled={disabled}
                className="h-8 text-xs"
            />
        </div>
    );
}
