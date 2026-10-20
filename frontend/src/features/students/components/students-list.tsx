"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import { MoreVertical } from "lucide-react";
import { DataTable } from "@/components/shared/data-table";
import { listStudents, bulkAssignStudentsToClass, type StudentListItem } from "@/lib/api/students";
import { useDeleteStudents } from "../hooks/use-students";
import { Button } from "@/components/ui/button";
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { studentFilterGroups, mapStudentFiltersToParams } from "./students-filters";
import { useClasses } from "@/features/classes/hooks/use-classes";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

function listStudentsWithFilters(params: {
    page?: number;
    limit?: number;
    search?: string;
    filters?: Record<string, string | string[]>;
}) {
    const { class_id, without_class, without_guardian } = mapStudentFiltersToParams(params.filters);
    return listStudents({
        page: params.page,
        limit: params.limit,
        search: params.search,
        class_id,
        without_class,
        without_guardian,
    });
}

export function StudentsTable() {
    const { mutateAsync: deleteStudents } = useDeleteStudents();
    const { data: classesResponse } = useClasses({ page: 1, limit: 100 });
    const queryClient = useQueryClient();
    const [assignClassId, setAssignClassId] = useState<string>("");
    const assignMutation = useMutation({
        mutationFn: async ({ classId, studentIds }: { classId: string; studentIds: string[] }) => {
            await bulkAssignStudentsToClass(classId, studentIds);
        },
        onSuccess: () => {
            toast.success("Students assigned to class");
            queryClient.invalidateQueries({ queryKey: ["students"] });
        },
        onError: () => {
            toast.error("Failed to assign students");
        },
    });

    const filterGroups = useMemo(
        () => [
            {
                id: "class",
                label: "Class",
                items: [
                    {
                        id: "class",
                        label: "Class",
                        type: "sub_menu_single" as const,
                        submenu:
                            classesResponse?.items?.map((c) => ({
                                id: c.id,
                                label: c.name,
                                value: c.id,
                            })) ?? [],
                    },
                ],
            },
            ...studentFilterGroups.filter((g) => g.id !== "class"),
        ],
        [classesResponse]
    );
    const columns = useMemo(
        () => [
            {
                id: "name",
                header: "Name",
                cell: (row: StudentListItem) => (
                    <Link
                        href={`/students/${row.student_id}`}
                        className="underline underline-offset-4 hover:no-underline"
                    >
                        {row.full_name}
                    </Link>
                ),
                width: "2fr",
            },
            {
                id: "admission_number",
                header: "Admission No.",
                cell: (row: StudentListItem) => row.admission_number,
                width: "1fr",
            },
            {
                id: "dob",
                header: "DOB",
                cell: (row: StudentListItem) => row.date_of_birth || "—",
                width: "1fr",
            },
            {
                id: "class",
                header: "Class",
                cell: (row: StudentListItem) =>
                    row.class_id ? (
                        <Link
                            href={`/classes/${row.class_id}`}
                            className="underline underline-offset-4 hover:no-underline"
                        >
                            {row.class_name || "—"}
                        </Link>
                    ) : (
                        row.class_name || "—"
                    ),
                width: "1fr",
            },
            {
                id: "grade_level",
                header: "Grade Level",
                cell: (row: StudentListItem) => row.grade_level || "—",
                width: "1fr",
            },
            {
                id: "gender",
                header: "Gender",
                cell: (row: StudentListItem) =>
                    row.gender === "F" || row.gender === "f"
                        ? "Female"
                        : row.gender === "M" || row.gender === "m"
                          ? "Male"
                          : row.gender || "—",
                width: "1fr",
            },
            {
                id: "guardian",
                header: "Guardian",
                cell: (row: StudentListItem) =>
                    row.guardian_linked ? (
                        <span className="inline-flex items-center rounded bg-green-100 px-2 py-0.5 text-xs font-medium text-green-800">
                            Linked
                        </span>
                    ) : (
                        <span className="inline-flex items-center rounded bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-600">
                            —
                        </span>
                    ),
                width: "1fr",
            },
            {
                id: "attendance",
                header: "Attendance %",
                cell: (row: StudentListItem) => {
                    const rate = row.attendance_rate;
                    if (rate == null) return "—";
                    return `${rate.toFixed(0)}%`;
                },
                width: "1fr",
            },
            {
                id: "actions",
                header: "",
                cell: (row: StudentListItem) => (
                    <DropdownMenu>
                        <DropdownMenuTrigger
                            render={
                                <Button variant="ghost" size="icon">
                                    <MoreVertical className="size-4" />
                                </Button>
                            }
                        />
                        <DropdownMenuContent align="end">
                            <DropdownMenuItem
                                render={<Link href={`/students/${row.student_id}`}>Edit</Link>}
                            />
                            <DropdownMenuItem
                                onSelect={() => {
                                    void deleteStudents([row.student_id]);
                                }}
                                className="text-destructive focus:text-destructive"
                            >
                                Delete
                            </DropdownMenuItem>
                        </DropdownMenuContent>
                    </DropdownMenu>
                ),
                width: "50px",
                align: "right" as const,
            },
        ],
        [deleteStudents]
    );

    return (
        <DataTable<
            StudentListItem,
            { filters?: Record<string, string | string[]> },
            { items: StudentListItem[]; total: number }
        >
            queryKey={["students"]}
            queryFn={listStudentsWithFilters}
            getRowId={(row) => row.student_id}
            columns={columns}
            isCheckable
            isSearchable
            searchPlaceholder="Search by name or admission number…"
            filterGroups={filterGroups}
            deleteFn={async (ids) => {
                await deleteStudents(ids);
            }}
            addHref="/students/add"
            pageSize={50}
            height={500}
            enableUrlSync
            urlSearchParam="search"
            urlFilterParamMap={{
                class: "class_id",
                without_class: "without_class",
                without_guardian: "without_guardian",
            }}
            renderToolBarComponents={(selectedIds) => {
                if (selectedIds.size === 0) return null;
                return (
                    <div className="flex items-center gap-2">
                        <Select value={assignClassId} onValueChange={setAssignClassId}>
                            <SelectTrigger className="w-[240px]">
                                <SelectValue placeholder="Select class" />
                            </SelectTrigger>
                            <SelectContent>
                                {classesResponse?.items?.map((c) => (
                                    <SelectItem key={c.id} value={c.id}>
                                        {c.name}
                                    </SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                        <Button
                            variant="default"
                            disabled={!assignClassId || assignMutation.isPending}
                            onClick={() => {
                                const ids = Array.from(selectedIds);
                                assignMutation.mutate({ classId: assignClassId, studentIds: ids });
                            }}
                        >
                            {assignMutation.isPending ? "Assigning…" : "Assign to class"}
                        </Button>
                    </div>
                );
            }}
        />
    );
}
