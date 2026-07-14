/**
 * TeacherAttendanceRoster — The teacher's view for marking attendance.
 *
 * Shows the roster for the current/next period with a RadioGroup for status,
 * optional note per row (via popover), and a batch submit button.
 * Renders empty state when no slot is active.
 */

"use client";

import { useState, useCallback, useMemo } from "react";
import Link from "next/link";
import { Loader2, Flag, StickyNote, ClipboardList } from "lucide-react";

import type { DataTableColumn } from "@/components/shared/data-table/types";
import { StaticTable } from "@/components/shared/static-table";
import { Button } from "@/components/ui/button";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Label } from "@/components/ui/label";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";

import type { RosterStudent } from "@/lib/api/attendance";
import { AttendanceEmptyState } from "./attendance-empty-state";
import { useSlotRoster, useBulkMarkAttendance } from "../hooks/use-attendance";
import type { AttendanceStatus } from "../types";

// ─── Props ────────────────────────────────────────────────────────────────

interface TeacherAttendanceRosterProps {
    timetableSlotId: string;
    date?: string;
    /** Whether this session is locked (past same-day window). */
    isLocked?: boolean;
}

// ─── Component ────────────────────────────────────────────────────────────

export function TeacherAttendanceRoster({
    timetableSlotId,
    date,
    isLocked = false,
}: TeacherAttendanceRosterProps) {
    const { data: roster, isLoading, isError } = useSlotRoster(timetableSlotId, date);
    const bulkMark = useBulkMarkAttendance();

    const [statusMap, setStatusMap] = useState<Record<string, AttendanceStatus>>({});
    const [noteMap, setNoteMap] = useState<Record<string, string>>({});

    // Initialise defaults from existing marks or "PRESENT"
    const effectiveStatus = useMemo(() => {
        if (!roster?.students) return statusMap;
        const map: Record<string, AttendanceStatus> = { ...statusMap };
        for (const student of roster.students) {
            if (!map[student.student_id]) {
                map[student.student_id] = student.current_status ?? "PRESENT";
            }
        }
        return map;
    }, [roster, statusMap]);

    const handleMarkAllPresent = useCallback(() => {
        if (!roster?.students) return;
        const allPresent: Record<string, AttendanceStatus> = {};
        for (const student of roster.students) {
            allPresent[student.student_id] = "PRESENT";
        }
        setStatusMap(allPresent);
    }, [roster]);

    const handleStatusChange = useCallback((studentId: string, status: AttendanceStatus) => {
        setStatusMap((prev) => ({ ...prev, [studentId]: status }));
    }, []);

    const handleNoteChange = useCallback((studentId: string, note: string) => {
        setNoteMap((prev) => ({ ...prev, [studentId]: note }));
    }, []);

    const handleSubmit = useCallback(() => {
        if (!roster?.students) return;
        const entries = roster.students.map((s) => ({
            student_id: s.student_id,
            status: effectiveStatus[s.student_id] ?? "PRESENT",
            note: noteMap[s.student_id] || null,
        }));
        bulkMark.mutate({
            timetable_slot_id: roster.timetable_slot_id,
            date: roster.date,
            entries,
        });
    }, [roster, effectiveStatus, noteMap, bulkMark]);

    // ── Columns (defined before early returns to keep hook order stable) ──

    const columns = useMemo<DataTableColumn<RosterStudent>[]>(
        () => [
            {
                id: "student",
                header: "Student",
                cell: (student) => <span className="font-medium">{student.full_name}</span>,
            },
            {
                id: "status",
                header: "Status",
                cell: (student) =>
                    isLocked ? (
                        <Badge
                            variant={
                                effectiveStatus[student.student_id] === "PRESENT"
                                    ? "default"
                                    : effectiveStatus[student.student_id] === "ABSENT"
                                      ? "destructive"
                                      : "secondary"
                            }
                        >
                            {{
                                PRESENT: "Present",
                                ABSENT: "Absent",
                                LATE: "Late",
                                EXCUSED: "Excused",
                            }[effectiveStatus[student.student_id]] ??
                                effectiveStatus[student.student_id]}
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
                                    />
                                    <Label
                                        htmlFor={`${student.student_id}-${option.value}`}
                                        className="cursor-pointer font-normal"
                                    >
                                        {option.label}
                                    </Label>
                                </div>
                            ))}
                        </RadioGroup>
                    ),
            },
            {
                id: "note",
                header: "Note",
                width: "100px",
                align: "center",
                cell: (student) => (
                    <Popover>
                        <PopoverTrigger asChild>
                            <Button
                                variant="ghost"
                                size="icon"
                                className="h-8 w-8"
                                aria-label="Add note"
                            >
                                <StickyNote className="h-4 w-4" />
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
                cell: (student) => (
                    <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8"
                        aria-label="Log behavior note"
                        asChild
                    >
                        <a
                            href={`/behavior/new?timetable_slot_id=${roster?.timetable_slot_id ?? ""}&student_id=${student.student_id}&date=${roster?.date ?? ""}`}
                        >
                            <Flag className="h-4 w-4" />
                        </a>
                    </Button>
                ),
            },
        ],
        [isLocked, effectiveStatus, handleStatusChange, handleNoteChange, noteMap, roster]
    );

    // ── Empty / Loading / Error states ────────────────────────────────────

    if (isLoading) {
        return (
            <div className="space-y-3">
                <Skeleton className="h-8 w-64" />
                {Array.from({ length: 8 }).map((_, i) => (
                    <Skeleton key={i} className="h-10 w-full" />
                ))}
            </div>
        );
    }

    if (isError) {
        return (
            <div className="border-destructive/50 text-destructive rounded-md border p-4">
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

    // ── Roster view ───────────────────────────────────────────────────────

    return (
        <div className="space-y-4">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div>
                    <h2 className="font-semibold">{roster.class_name}</h2>
                    <p className="text-muted-foreground">
                        {roster.learning_area} &middot; {roster.date}
                    </p>
                </div>
                {!isLocked && (
                    <Button variant="outline" size="sm" onClick={handleMarkAllPresent}>
                        Mark all Present
                    </Button>
                )}
            </div>

            {/* Table */}
            <StaticTable
                columns={columns}
                data={roster.students}
                getRowId={(student) => student.student_id}
                rowHeight={48}
                height={334}
            />

            {/* Submit or Locked notice */}
            {isLocked ? (
                <p className="text-muted-foreground">
                    Locked &mdash; contact your admin for corrections.
                </p>
            ) : (
                <div className="flex items-center gap-3">
                    <Button onClick={handleSubmit} disabled={bulkMark.isPending} size="lg">
                        {bulkMark.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                        Save Attendance
                    </Button>
                    {bulkMark.isSuccess && (
                        <span className="text-emerald-600">
                            Saved &mdash; {bulkMark.data?.count ?? 0} students marked
                        </span>
                    )}
                </div>
            )}
        </div>
    );
}
