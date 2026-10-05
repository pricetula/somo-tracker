"use client";

import { useMemo } from "react";
import { MoreVertical } from "lucide-react";
import Link from "next/link";
import { DataTable } from "@/components/shared/data-table";
import { listByClass } from "../services/api";
import { useDeleteEnrollment } from "../hooks/use-enrollments";
import type { Enrollment } from "../types/enrollment";
import { Button } from "@/components/ui/button";
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
export function EnrollmentsTable({ classId }: { classId: string }) {
    const { mutateAsync: deleteEnrollment } = useDeleteEnrollment(classId);

    const columns = useMemo(
        () => [
            {
                id: "student_name",
                header: "Student",
                cell: (row: Enrollment) => row.student_name ?? "—",
                width: "2fr",
            },
            {
                id: "admission_number",
                header: "Admission",
                cell: (row: Enrollment) => row.admission_number ?? "—",
                width: "1fr",
            },
            {
                id: "class_name",
                header: "Class",
                cell: (row: Enrollment) => row.class_name ?? "—",
                width: "1fr",
            },
            {
                id: "status",
                header: "Status",
                cell: (row: Enrollment) => row.status,
                width: "1fr",
            },

            {
                id: "actions",
                header: "",
                cell: (row: Enrollment) => (
                    <DropdownMenu>
                        <DropdownMenuTrigger
                            render={
                                <Button variant="ghost" size="icon">
                                    <MoreVertical className="size-4" />
                                </Button>
                            }
                        />
                        <DropdownMenuContent align="end">
                            <DropdownMenuItem>
                                <Link href={`#`}>Edit</Link>
                            </DropdownMenuItem>
                            <DropdownMenuItem
                                onClick={() => {
                                    void deleteEnrollment(row.id);
                                }}
                                className="text-destructive focus:text-destructive"
                            >
                                Unenroll
                            </DropdownMenuItem>
                        </DropdownMenuContent>
                    </DropdownMenu>
                ),
                width: "50px",
                align: "right" as const,
            },
        ],
        [deleteEnrollment]
    );

    return (
        <DataTable<
            Enrollment,
            { filters?: Record<string, string | string[]> },
            { items: Enrollment[]; total: number }
        >
            queryKey={["enrollments", classId]}
            queryFn={(params) => listByClass(classId, params)}
            getRowId={(row) => row.id}
            columns={columns}
            isCheckable
            pageSize={50}
            height={500}
        />
    );
}
