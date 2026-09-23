"use client";
import Link from "next/link";
import { DataTable } from "@/components/shared/data-table";
import { listAttendanceSessions } from "../services/api";
import { StatusBadge } from "./status-badge";
import type { AttendanceSession, ListAttendanceSessionsParams } from "../types/attendance";
import { useGrades } from "@/features/grades/hooks/use-grades";
import { useStreams } from "@/features/streams/hooks/use-streams";
import { useSubjects } from "@/features/curriculum/hooks/use-subjects";
import { useTeachers } from "@/features/teachers/hooks/use-teachers-list";
import { useMemo } from "react";
import type { FilterGroup, DataTableColumn } from "@/components/shared/data-table/types";

function listWithFilters(params: ListAttendanceSessionsParams) {
    return listAttendanceSessions(params);
}

interface FilterOption {
    id: string;
    label: string;
    value: string;
}

export function AttendanceTable() {
    const { data: grades = [] } = useGrades();
    const { data: streams = [] } = useStreams();
    const subjectsResult = useSubjects({ limit: 100 });
    const teachersResult = useTeachers({ limit: 100 });

    const gradeOptions = useMemo<FilterOption[]>(
        () => grades?.map((g) => ({ id: g.id, label: g.local_label, value: g.local_label })) ?? [],
        [grades]
    );
    const streamOptions = useMemo<FilterOption[]>(
        () => streams?.map((s) => ({ id: s.id, label: s.name, value: s.name })) ?? [],
        [streams]
    );
    const subjectOptions = useMemo<FilterOption[]>(
        () =>
            subjectsResult.data?.items?.map((s) => ({ id: s.id, label: s.name, value: s.name })) ??
            [],
        [subjectsResult]
    );
    const teacherOptions = useMemo<FilterOption[]>(
        () =>
            teachersResult.data?.items?.map((t) => ({
                id: t.user_id,
                label: t.full_name,
                value: t.full_name,
            })) ?? [],
        [teachersResult]
    );

    const filterGroups = useMemo<FilterGroup[]>(
        () => [
            {
                id: "status",
                label: "Status",
                items: [
                    {
                        id: "status-filter",
                        label: "Status",
                        type: "sub_menu_multi" as const,
                        submenu: [
                            { id: "SUBMITTED", label: "Submitted", value: "SUBMITTED" },
                            { id: "IN_PROGRESS", label: "In Progress", value: "IN_PROGRESS" },
                            { id: "MISSED", label: "Missed", value: "MISSED" },
                        ],
                    },
                ],
            },
            {
                id: "grade",
                label: "Grade",
                items: [
                    {
                        id: "grade-filter",
                        label: "Grade",
                        type: "sub_menu_multi" as const,
                        submenu:
                            gradeOptions.length > 0
                                ? gradeOptions.map((opt) => ({
                                      id: opt.id,
                                      label: opt.label,
                                      value: opt.value,
                                  }))
                                : [{ id: "empty", label: "No grades", value: "" }],
                    },
                ],
            },
            {
                id: "stream",
                label: "Stream",
                items: [
                    {
                        id: "stream-filter",
                        label: "Stream",
                        type: "sub_menu_multi" as const,
                        submenu:
                            streamOptions.length > 0
                                ? streamOptions.map((opt) => ({
                                      id: opt.id,
                                      label: opt.label,
                                      value: opt.value,
                                  }))
                                : [{ id: "empty", label: "No streams", value: "" }],
                    },
                ],
            },
            {
                id: "subject",
                label: "Subject",
                items: [
                    {
                        id: "subject-filter",
                        label: "Subject",
                        type: "sub_menu_multi" as const,
                        submenu:
                            subjectOptions.length > 0
                                ? subjectOptions.map((opt) => ({
                                      id: opt.id,
                                      label: opt.label,
                                      value: opt.value,
                                  }))
                                : [{ id: "empty", label: "No subjects", value: "" }],
                    },
                ],
            },
            {
                id: "teacher",
                label: "Teacher",
                items: [
                    {
                        id: "teacher-filter",
                        label: "Teacher",
                        type: "sub_menu_multi" as const,
                        submenu:
                            teacherOptions.length > 0
                                ? teacherOptions.map((opt) => ({
                                      id: opt.id,
                                      label: opt.label,
                                      value: opt.value,
                                  }))
                                : [{ id: "empty", label: "No teachers", value: "" }],
                    },
                ],
            },
        ],
        [gradeOptions, streamOptions, subjectOptions, teacherOptions]
    );

    const columns = useMemo<DataTableColumn<AttendanceSession>[]>(
        () => [
            {
                id: "className",
                header: "Class",
                cell: (row: AttendanceSession) => (
                    <Link
                        href={`/classes/${row.classId}`}
                        className="underline underline-offset-4 hover:no-underline"
                    >
                        {row.className}
                    </Link>
                ),
                width: "2fr",
            },
            {
                id: "teacherName",
                header: "Teacher",
                cell: (row: AttendanceSession) => (
                    <Link
                        href={`/teachers/${row.teacherId}`}
                        className="underline underline-offset-4 hover:no-underline"
                    >
                        {row.teacherName}
                    </Link>
                ),
                width: "1.5fr",
            },
            {
                id: "subject",
                header: "Subject",
                cell: (row: AttendanceSession) => row.subject,
                width: "1.5fr",
            },
            {
                id: "dateTime",
                header: "Date & Time",
                cell: (row: AttendanceSession) => (
                    <div className="space-y-0.5">
                        <span>{row.attendanceDate}</span>
                        <span className="text-muted-foreground text-[0.625rem]">
                            {row.timeSlotName} · {row.startTime}–{row.endTime}
                        </span>
                    </div>
                ),
                width: "1.5fr",
            },
            {
                id: "status",
                header: "Status",
                cell: (row: AttendanceSession) => <StatusBadge status={row.status} />,
                width: "1fr",
                align: "center",
            },
            {
                id: "progress",
                header: "Progress",
                cell: (row: AttendanceSession) => (
                    <div className="flex items-center gap-2">
                        <div className="bg-muted h-1.5 max-w-32 flex-1 overflow-hidden rounded-full">
                            <div
                                className="bg-primary h-full transition-all"
                                style={{
                                    width: `${row.totalStudents > 0 ? (row.recordedStudents / row.totalStudents) * 100 : 0}%`,
                                }}
                            />
                        </div>
                        <span className="text-muted-foreground w-20 text-right text-xs">
                            {row.recordedStudents}/{row.totalStudents}
                        </span>
                    </div>
                ),
                width: "1.5fr",
            },
        ],
        []
    );

    return (
        <DataTable<
            AttendanceSession,
            ListAttendanceSessionsParams,
            { items: AttendanceSession[]; total: number; page: number; limit: number }
        >
            queryKey={["attendance", "sessions"]}
            queryFn={listWithFilters}
            getRowId={(row) => row.slotId + "-" + row.attendanceDate}
            columns={columns}
            isSearchable
            searchPlaceholder="Search class, teacher, subject…"
            filterGroups={filterGroups}
            pageSize={50}
            height={500}
        />
    );
}
