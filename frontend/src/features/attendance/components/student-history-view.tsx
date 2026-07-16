/**
 * StudentHistoryView — raw period-by-period attendance history for a student.
 *
 * Used by admins for manual interpretation. Includes term filter and inline edit.
 */

"use client";

import { useState } from "react";
import { CalendarX } from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import { Card, CardContent } from "@/components/ui/card";
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Loader2, Pencil } from "lucide-react";

import { AttendanceEmptyState } from "./attendance-empty-state";
import { useStudentHistory, useUpdateAttendanceRecord } from "../hooks/use-attendance";
import { useAcademicTerms } from "@/features/academic-terms/hooks/use-academic-terms";
import type { AttendanceStatus } from "../types";
import { attendanceBadgeProps, attendanceStatusLabel } from "../types";

interface StudentHistoryViewProps {
    studentId: string;
    termId?: string;
}

export function StudentHistoryView({ studentId, termId }: StudentHistoryViewProps) {
    const [selectedTermId, setSelectedTermId] = useState(termId ?? "");
    const [startDate, setStartDate] = useState("");
    const [endDate, setEndDate] = useState("");
    const { data: termsData } = useAcademicTerms();
    const terms = termsData?.items ?? [];
    const [editingId, setEditingId] = useState<string | null>(null);
    const [editStatus, setEditStatus] = useState<AttendanceStatus>("PRESENT");

    const { data, isLoading, isError } = useStudentHistory(studentId, {
        term_id: selectedTermId,
        start_date: startDate || undefined,
        end_date: endDate || undefined,
    });
    const updateRecord = useUpdateAttendanceRecord();

    const handleEdit = (recordId: string, currentStatus: AttendanceStatus) => {
        setEditingId(recordId);
        setEditStatus(currentStatus);
    };

    const handleSave = (recordId: string) => {
        updateRecord.mutate(
            { recordId, status: editStatus },
            {
                onSuccess: () => {
                    setEditingId(null);
                },
            }
        );
    };

    if (isLoading) {
        return (
            <div className="space-y-3">
                <Skeleton className="h-8 w-48" />
                <Skeleton className="h-8 w-full" />
                <Skeleton className="h-8 w-full" />
                <Skeleton className="h-8 w-full" />
            </div>
        );
    }

    if (isError) {
        return (
            <div className="border-destructive/50 text-destructive rounded-md border p-4">
                Failed to load attendance history.
            </div>
        );
    }

    const records = data?.items ?? [];

    // Compute summary from records
    const summary = (() => {
        if (!records.length) return null;
        const counts: Record<string, number> = {};
        for (const r of records) {
            counts[r.status] = (counts[r.status] || 0) + 1;
        }
        const total = records.length;
        const present = counts.PRESENT || 0;
        const percentage = total > 0 ? Math.round((present / total) * 100 * 10) / 10 : 0;
        return {
            total,
            present,
            absent: counts.ABSENT || 0,
            late: counts.LATE || 0,
            excused: counts.EXCUSED || 0,
            percentage,
        };
    })();

    return (
        <div className="space-y-4">
            {/* Summary card */}
            {summary && (
                <Card size="sm">
                    <CardContent className="flex items-center gap-6 py-3">
                        <div className="flex items-baseline gap-1">
                            <span className="text-2xl font-bold">{summary.percentage}%</span>
                            <span className="text-muted-foreground text-xs">attendance</span>
                        </div>
                        <div className="flex flex-wrap gap-1.5">
                            {(["PRESENT", "ABSENT", "LATE", "EXCUSED"] as const)
                                .filter((s) => summary[s.toLowerCase() as keyof typeof summary] > 0)
                                .map((s) => (
                                    <Badge
                                        key={s}
                                        variant={attendanceBadgeProps(s).variant}
                                        className={attendanceBadgeProps(s).className}
                                    >
                                        {attendanceStatusLabel(s)}:{" "}
                                        {summary[s.toLowerCase() as keyof typeof summary]}
                                    </Badge>
                                ))}
                        </div>
                        <span className="text-muted-foreground ml-auto text-xs">
                            {summary.total} total periods
                        </span>
                    </CardContent>
                </Card>
            )}

            {/* Filters */}
            <div className="flex flex-wrap items-center gap-3">
                <Select value={selectedTermId} onValueChange={setSelectedTermId}>
                    <SelectTrigger className="w-44">
                        <SelectValue placeholder="Select term" />
                    </SelectTrigger>
                    <SelectContent>
                        <SelectItem value="">All terms</SelectItem>
                        {terms.map((term) => (
                            <SelectItem key={term.id} value={term.id}>
                                {term.name}
                                {term.is_current ? " (current)" : ""}
                            </SelectItem>
                        ))}
                    </SelectContent>
                </Select>
                <div className="flex items-center gap-2">
                    <Label className="text-muted-foreground text-xs">From</Label>
                    <Input
                        type="date"
                        value={startDate}
                        onChange={(e) => setStartDate(e.target.value)}
                        className="w-40"
                    />
                </div>
                <div className="flex items-center gap-2">
                    <Label className="text-muted-foreground text-xs">To</Label>
                    <Input
                        type="date"
                        value={endDate}
                        onChange={(e) => setEndDate(e.target.value)}
                        className="w-40"
                    />
                </div>
                <p className="text-muted-foreground">{records.length} records</p>
                {(startDate || endDate) && (
                    <Button
                        variant="ghost"
                        size="sm"
                        className="text-xs"
                        onClick={() => {
                            setStartDate("");
                            setEndDate("");
                        }}
                    >
                        Clear dates
                    </Button>
                )}
            </div>

            {records.length === 0 ? (
                <AttendanceEmptyState
                    icon={CalendarX}
                    title="No attendance records found"
                    description="No attendance marks have been recorded for this student matching the current filters."
                >
                    {selectedTermId && (
                        <Button variant="outline" size="sm" onClick={() => setSelectedTermId("")}>
                            Show all terms
                        </Button>
                    )}
                </AttendanceEmptyState>
            ) : (
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead>Date</TableHead>
                            <TableHead>Status</TableHead>
                            <TableHead>Note</TableHead>
                            <TableHead className="w-20" />
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {records.map((record) => (
                            <TableRow key={record.id}>
                                <TableCell>{record.date}</TableCell>
                                <TableCell>
                                    {editingId === record.id ? (
                                        <Select
                                            value={editStatus}
                                            onValueChange={(val) =>
                                                setEditStatus(val as AttendanceStatus)
                                            }
                                        >
                                            <SelectTrigger className="w-32">
                                                <SelectValue />
                                            </SelectTrigger>
                                            <SelectContent>
                                                <SelectItem value="PRESENT">Present</SelectItem>
                                                <SelectItem value="ABSENT">Absent</SelectItem>
                                                <SelectItem value="LATE">Late</SelectItem>
                                                <SelectItem value="EXCUSED">Excused</SelectItem>
                                            </SelectContent>
                                        </Select>
                                    ) : (
                                        <Badge
                                            variant={attendanceBadgeProps(record.status).variant}
                                            className={
                                                attendanceBadgeProps(record.status).className
                                            }
                                        >
                                            {attendanceStatusLabel(record.status)}
                                        </Badge>
                                    )}
                                </TableCell>
                                <TableCell className="text-muted-foreground">
                                    {record.note ?? "—"}
                                </TableCell>
                                <TableCell>
                                    {editingId === record.id ? (
                                        <Button
                                            size="sm"
                                            onClick={() => handleSave(record.id)}
                                            disabled={updateRecord.isPending}
                                        >
                                            {updateRecord.isPending && (
                                                <Loader2 className="mr-1 h-3 w-3 animate-spin" />
                                            )}
                                            Save
                                        </Button>
                                    ) : (
                                        <Button
                                            variant="ghost"
                                            size="icon"
                                            className="h-8 w-8"
                                            onClick={() => handleEdit(record.id, record.status)}
                                        >
                                            <Pencil className="h-4 w-4" />
                                        </Button>
                                    )}
                                </TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            )}
        </div>
    );
}
