"use client";
import { useMemo, useCallback } from "react";
import Link from "next/link";
import { DataTable } from "@/components/shared/data-table";
import { listSubstitutions } from "../services/substitutions-api";
import { useSubstitutions } from "../hooks/use-substitutions";
import { SubstitutionForm } from "./substitution-form";
import type { TimetableSubstitution, ListSubstitutionsParams } from "../types/substitution";
import { useDeleteSubstitution } from "../hooks/use-substitutions";
import { MoreVertical } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

function listWithFilters(params: ListSubstitutionsParams) {
    return listSubstitutions(params);
}

export function SubstitutionsTable() {
    const { mutateAsync: deleteSub } = useDeleteSubstitution();
    const handleDelete = useCallback(
        async (ids: string[]) => {
            await deleteSub(ids[0]);
        },
        [deleteSub]
    );

    const filterGroups = useMemo(
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
                            { id: "PENDING", label: "Pending", value: "PENDING" },
                            { id: "ASSIGNED", label: "Assigned", value: "ASSIGNED" },
                            { id: "COMPLETED", label: "Completed", value: "COMPLETED" },
                            { id: "CANCELLED", label: "Cancelled", value: "CANCELLED" },
                        ],
                    },
                ],
            },
        ],
        []
    );

    const columns = useMemo(
        () => [
            {
                id: "className",
                header: "Class",
                cell: (row: TimetableSubstitution) => (
                    <Link
                        href={`/classes/${row.classTimetableSlotId}`}
                        className="underline underline-offset-4 hover:no-underline"
                    >
                        {row.className}
                    </Link>
                ),
                width: "2fr",
            },
            {
                id: "subjectName",
                header: "Subject",
                cell: (row: TimetableSubstitution) => row.subjectName,
                width: "1.5fr",
            },
            {
                id: "substitutionDate",
                header: "Date",
                cell: (row: TimetableSubstitution) => row.substitutionDate,
                width: "1fr",
                align: "center" as const,
            },
            {
                id: "timeSlot",
                header: "Time",
                cell: (row: TimetableSubstitution) => (
                    <div>
                        <span>{row.timeSlotName}</span>
                        <span className="text-muted-foreground ml-2 text-xs">
                            {row.startTime}–{row.endTime}
                        </span>
                    </div>
                ),
                width: "1.5fr",
            },
            {
                id: "originalTeacher",
                header: "Original Teacher",
                cell: (row: TimetableSubstitution) => row.originalTeacherName,
                width: "1.5fr",
            },
            {
                id: "substituteTeacher",
                header: "Substitute",
                cell: (row: TimetableSubstitution) => row.substituteTeacherName ?? "—",
                width: "1.5fr",
            },
            {
                id: "status",
                header: "Status",
                cell: (row: TimetableSubstitution) => {
                    const statusConfig: Record<string, { label: string; className: string }> = {
                        PENDING: { label: "Pending", className: "bg-amber-100 text-amber-800" },
                        ASSIGNED: { label: "Assigned", className: "bg-blue-100 text-blue-800" },
                        COMPLETED: {
                            label: "Completed",
                            className: "bg-emerald-100 text-emerald-800",
                        },
                        CANCELLED: { label: "Cancelled", className: "bg-rose-100 text-rose-800" },
                    };
                    const { label, className } = statusConfig[row.status] || {
                        label: row.status,
                        className: "bg-muted text-muted-foreground",
                    };
                    return (
                        <span
                            className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${className}`}
                        >
                            {label}
                        </span>
                    );
                },
                width: "1fr",
                align: "center" as const,
            },
            {
                id: "actions",
                header: "",
                cell: (row: TimetableSubstitution) => (
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
                                render={
                                    <Link href={`/timetable/substitutions/${row.id}`}>Edit</Link>
                                }
                            />
                            <DropdownMenuItem
                                onSelect={() => handleDelete([row.id])}
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
        []
    );

    return (
        <DataTable<
            TimetableSubstitution,
            ListSubstitutionsParams,
            { items: TimetableSubstitution[]; total: number; page: number; limit: number }
        >
            queryKey={["timetable", "substitutions", "list"]}
            queryFn={listWithFilters}
            getRowId={(row) => row.id}
            columns={columns}
            isSearchable
            searchPlaceholder="Search class, teacher, subject…"
            filterGroups={filterGroups}
            addHref="/timetable/substitutions/add"
            pageSize={50}
            height={500}
            deleteFn={async (ids) => {
                await deleteSub(ids[0]);
            }}
        />
    );
}
