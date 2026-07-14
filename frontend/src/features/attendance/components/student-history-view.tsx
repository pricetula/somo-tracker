/**
 * StudentHistoryView — raw period-by-period attendance history for a student.
 *
 * Used by admins for manual interpretation. Includes term filter and inline edit.
 */

"use client";

import { useState } from "react";
import { Skeleton } from "@/components/ui/skeleton";
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
import { Loader2, Pencil } from "lucide-react";

import { useStudentHistory, useUpdateAttendanceRecord } from "../hooks/use-attendance";
import type { AttendanceStatus } from "../types";

interface StudentHistoryViewProps {
    studentId: string;
    termId?: string;
}

export function StudentHistoryView({ studentId, termId }: StudentHistoryViewProps) {
    const [selectedTermId, setSelectedTermId] = useState(termId ?? "");
    const [editingId, setEditingId] = useState<string | null>(null);
    const [editStatus, setEditStatus] = useState<AttendanceStatus>("PRESENT");

    const { data, isLoading, isError } = useStudentHistory(studentId, {
        term_id: selectedTermId,
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

    return (
        <div className="space-y-4">
            <div className="flex items-center gap-3">
                <Select value={selectedTermId} onValueChange={setSelectedTermId}>
                    <SelectTrigger className="w-48">
                        <SelectValue placeholder="Select term" />
                    </SelectTrigger>
                    <SelectContent>
                        <SelectItem value="">All terms</SelectItem>
                        {/* Terms populated by parent component or via combobox */}
                    </SelectContent>
                </Select>
                <p className="text-muted-foreground text-sm">{records.length} records</p>
            </div>

            {records.length === 0 ? (
                <p className="text-muted-foreground py-8 text-center">
                    No attendance records found.
                </p>
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
                                            variant={
                                                record.status === "PRESENT"
                                                    ? "default"
                                                    : record.status === "ABSENT"
                                                      ? "destructive"
                                                      : "secondary"
                                            }
                                        >
                                            {record.status}
                                        </Badge>
                                    )}
                                </TableCell>
                                <TableCell className="text-muted-foreground text-sm">
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
