/**
 * TeacherAttendanceRoster — The teacher's view for marking attendance.
 *
 * Shows the roster for the current/next period with a ToggleGroup for status,
 * optional note per row (via popover), and a batch submit button.
 * Renders empty state when no slot is active.
 */

"use client";

import { useState, useCallback, useMemo } from "react";
import { Loader2, Flag, StickyNote } from "lucide-react";

import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";

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
        if (!roster) return statusMap;
        const map: Record<string, AttendanceStatus> = { ...statusMap };
        for (const student of roster.students) {
            if (!map[student.student_id]) {
                map[student.student_id] = student.current_status ?? "PRESENT";
            }
        }
        return map;
    }, [roster, statusMap]);

    const handleMarkAllPresent = useCallback(() => {
        if (!roster) return;
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
        if (!roster) return;
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

    if (!roster || roster.students.length === 0) {
        return (
            <div className="text-muted-foreground flex items-center justify-center py-16">
                <p>No class in session right now.</p>
            </div>
        );
    }

    // ── Roster view ───────────────────────────────────────────────────────

    return (
        <div className="space-y-4">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div>
                    <h2 className="text-xl font-semibold">{roster.class_name}</h2>
                    <p className="text-muted-foreground text-sm">
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
            <Table>
                <TableHeader>
                    <TableRow>
                        <TableHead className="w-12">#</TableHead>
                        <TableHead>Student</TableHead>
                        <TableHead className="w-28">Adm No.</TableHead>
                        <TableHead className="w-80">Status</TableHead>
                        <TableHead className="w-12">Note</TableHead>
                        <TableHead className="w-10" />
                    </TableRow>
                </TableHeader>
                <TableBody>
                    {roster.students.map((student, idx) => (
                        <TableRow key={student.student_id}>
                            <TableCell className="text-muted-foreground">{idx + 1}</TableCell>
                            <TableCell className="font-medium">{student.full_name}</TableCell>
                            <TableCell className="text-muted-foreground">
                                {student.admission_number}
                            </TableCell>
                            <TableCell>
                                {isLocked ? (
                                    <Badge
                                        variant={
                                            effectiveStatus[student.student_id] === "PRESENT"
                                                ? "default"
                                                : effectiveStatus[student.student_id] === "ABSENT"
                                                  ? "destructive"
                                                  : "secondary"
                                        }
                                    >
                                        {effectiveStatus[student.student_id]}
                                    </Badge>
                                ) : (
                                    <ToggleGroup
                                        type="single"
                                        value={effectiveStatus[student.student_id] ?? "PRESENT"}
                                        onValueChange={(val) => {
                                            if (val)
                                                handleStatusChange(
                                                    student.student_id,
                                                    val as AttendanceStatus
                                                );
                                        }}
                                        size="sm"
                                    >
                                        <ToggleGroupItem value="PRESENT" aria-label="Present">
                                            P
                                        </ToggleGroupItem>
                                        <ToggleGroupItem value="ABSENT" aria-label="Absent">
                                            A
                                        </ToggleGroupItem>
                                        <ToggleGroupItem value="LATE" aria-label="Late">
                                            L
                                        </ToggleGroupItem>
                                        <ToggleGroupItem value="EXCUSED" aria-label="Excused">
                                            E
                                        </ToggleGroupItem>
                                    </ToggleGroup>
                                )}
                            </TableCell>
                            <TableCell>
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
                            </TableCell>
                            <TableCell>
                                {/* Flag icon for behavior note — links to behavior quick-add */}
                                <Button
                                    variant="ghost"
                                    size="icon"
                                    className="h-8 w-8"
                                    aria-label="Log behavior note"
                                    asChild
                                >
                                    <a
                                        href={`/behavior/new?timetable_slot_id=${roster.timetable_slot_id}&student_id=${student.student_id}&date=${roster.date}`}
                                    >
                                        <Flag className="h-4 w-4" />
                                    </a>
                                </Button>
                            </TableCell>
                        </TableRow>
                    ))}
                </TableBody>
            </Table>

            {/* Submit or Locked notice */}
            {isLocked ? (
                <p className="text-muted-foreground text-sm">
                    Locked &mdash; contact your admin for corrections.
                </p>
            ) : (
                <div className="flex items-center gap-3">
                    <Button onClick={handleSubmit} disabled={bulkMark.isPending} size="lg">
                        {bulkMark.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                        Save Attendance
                    </Button>
                    {bulkMark.isSuccess && (
                        <span className="text-sm text-green-600">
                            Saved &mdash; {bulkMark.data?.count ?? 0} students marked
                        </span>
                    )}
                </div>
            )}
        </div>
    );
}
