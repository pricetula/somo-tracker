/**
 * StudentHistoryView — raw period-by-period attendance history for a student.
 * Pure shadcn: no cards/borders, no hardcoded colours.
 */

"use client";

import { useState } from "react";
import { CalendarX, Loader2, Pencil } from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import {
    Dialog,
    DialogContent,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";
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
import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";

import { AttendanceEmptyState } from "./attendance-empty-state";
import { useStudentHistory, useUpdateAttendanceRecord } from "../hooks/use-attendance";
import { useAcademicTerms } from "@/features/academic-terms/hooks/use-academic-terms";
import type { AttendanceStatus, AttendanceRecord } from "../types";
import { attendanceBadgeProps, attendanceBadgeClass, attendanceStatusLabel } from "../types";

function StatusBadgeCell({ status }: { status: AttendanceStatus }) {
    return (
        <Badge
            variant={attendanceBadgeProps(status).variant}
            className={attendanceBadgeClass(status)}
        >
            {attendanceStatusLabel(status)}
        </Badge>
    );
}

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

    const [editRecord, setEditRecord] = useState<AttendanceRecord | null>(null);
    const [editStatus, setEditStatus] = useState<AttendanceStatus>("PRESENT");
    const updateRecord = useUpdateAttendanceRecord();

    const { data, isLoading, isError } = useStudentHistory(studentId, {
        term_id: selectedTermId,
        start_date: startDate || undefined,
        end_date: endDate || undefined,
    });

    const records = data?.items ?? [];

    const queryFn = () => Promise.resolve({ items: records, total: records.length });

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

    const openEdit = (record: AttendanceRecord) => {
        setEditRecord(record);
        setEditStatus(record.status);
    };

    const handleSaveEdit = () => {
        if (!editRecord) return;
        updateRecord.mutate(
            { recordId: editRecord.id, status: editStatus },
            { onSuccess: () => setEditRecord(null) }
        );
    };

    const columns: DataTableColumn<AttendanceRecord>[] = [
        {
            id: "date",
            header: "Date",
            cell: (row) => <span className="text-foreground">{row.date}</span>,
        },
        {
            id: "status",
            header: "Status",
            width: "120px",
            cell: (row) => <StatusBadgeCell status={row.status} />,
        },
        {
            id: "note",
            header: "Note",
            width: "minmax(120px, 1fr)",
            cell: (row) => <span className="text-muted-foreground">{row.note ?? "\u2014"}</span>,
        },
        {
            id: "actions",
            header: "",
            width: "60px",
            align: "right",
            cell: (row) => (
                <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8"
                    onClick={() => openEdit(row)}
                >
                    <Pencil className="h-4 w-4" />
                </Button>
            ),
        },
    ];

    if (isLoading) {
        return (
            <div className="space-y-3">
                <Skeleton className="h-8 w-48" />
                <Skeleton className="h-8 w-full" />
                <Skeleton className="h-8 w-full" />
            </div>
        );
    }

    if (isError) {
        return (
            <div className="text-destructive bg-destructive/10 p-4">
                Failed to load attendance history.
            </div>
        );
    }

    return (
        <div className="space-y-4">
            {/* Summary — no Card, just subtle background */}
            {summary && (
                <div className="bg-muted/30 flex items-center gap-6 p-3">
                    <div className="flex items-baseline gap-1">
                        <span className="text-foreground text-2xl font-bold">
                            {summary.percentage}%
                        </span>
                        <span className="text-muted-foreground text-xs">attendance</span>
                    </div>
                    <div className="flex flex-wrap gap-1.5">
                        {(["PRESENT", "ABSENT", "LATE", "EXCUSED"] as const)
                            .filter((s) => summary[s.toLowerCase() as keyof typeof summary] > 0)
                            .map((s) => (
                                <Badge key={s} variant={attendanceBadgeProps(s).variant}>
                                    {attendanceStatusLabel(s)}:{" "}
                                    {summary[s.toLowerCase() as keyof typeof summary]}
                                </Badge>
                            ))}
                    </div>
                    <span className="text-muted-foreground ml-auto text-xs">
                        {summary.total} total periods
                    </span>
                </div>
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
                <DataTable
                    queryKey={[
                        "attendance",
                        "history",
                        studentId,
                        selectedTermId,
                        startDate,
                        endDate,
                    ]}
                    queryFn={queryFn}
                    columns={columns}
                    getRowId={(row) => row.id}
                    height={Math.min(records.length * 44 + 50, 500)}
                    pageSize={100}
                    emptyState="No attendance records found."
                    noResultsState="No records match your filters."
                />
            )}

            <Dialog
                open={!!editRecord}
                onOpenChange={(open) => {
                    if (!open) setEditRecord(null);
                }}
            >
                <DialogContent className="sm:max-w-sm">
                    <DialogHeader>
                        <DialogTitle>Edit Attendance Record</DialogTitle>
                    </DialogHeader>
                    <div className="space-y-2">
                        <Label className="text-foreground">Status</Label>
                        <Select
                            value={editStatus}
                            onValueChange={(val) => setEditStatus(val as AttendanceStatus)}
                        >
                            <SelectTrigger>
                                <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                                <SelectItem value="PRESENT">Present</SelectItem>
                                <SelectItem value="ABSENT">Absent</SelectItem>
                                <SelectItem value="LATE">Late</SelectItem>
                                <SelectItem value="EXCUSED">Excused</SelectItem>
                            </SelectContent>
                        </Select>
                    </div>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setEditRecord(null)}>
                            Cancel
                        </Button>
                        <Button onClick={handleSaveEdit} disabled={updateRecord.isPending}>
                            {updateRecord.isPending && (
                                <Loader2 className="mr-1 h-3 w-3 animate-spin" />
                            )}
                            Save
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </div>
    );
}
