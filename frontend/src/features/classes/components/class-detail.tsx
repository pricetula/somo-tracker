"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { getClass } from "../services/api";
import type { ClassDetail } from "../types/class";
import { DataTable } from "@/components/shared/data-table/data-table";
import { listByClass } from "@/features/enrollments/services/api";

interface ClassDetailProps {
    id: string;
}

export function ClassDetail({ id }: ClassDetailProps) {
    const { data: detail, isLoading } = useQuery({
        queryKey: ["class", id],
        queryFn: () => getClass(id),
    });

    if (isLoading) return <div className="p-6">Loading...</div>;
    if (!detail) return null;

    return (
        <div className="space-y-6 p-6">
            <div className="flex items-start justify-between gap-4">
                <div className="space-y-1">
                    <h1 className="text-2xl font-semibold">{detail.name}</h1>
                    <p className="text-muted-foreground">
                        {detail.grade} • {detail.stream} • {detail.academicYear}
                    </p>
                </div>
                <Link
                    href={`/classes/${id}/enrollments`}
                    className="bg-primary text-primary-foreground hover:bg-primary/90 inline-flex items-center rounded-md px-3 py-1.5 text-sm font-medium"
                >
                    Enroll Students
                </Link>
            </div>
            <div className="space-y-2">
                <div>
                    <span className="font-medium">Teacher:</span> {detail.teacherName ?? "—"}
                </div>
                <div>
                    <span className="font-medium">Students:</span> {detail.studentsCount}
                </div>
                {detail.description && (
                    <div>
                        <span className="font-medium">Description:</span> {detail.description}
                    </div>
                )}
            </div>

            <div className="space-y-4">
                <h2 className="text-lg font-semibold">Enrolled Students</h2>
                <DataTable<
                    {
                        id: string;
                        student_id: string;
                        student_name?: string;
                        admission_number?: string;
                        status?: string;
                        enrolled_at?: string;
                    },
                    Record<string, never>,
                    { items: unknown[]; total: number }
                >
                    queryKey={["enrollments", id]}
                    queryFn={(params) =>
                        listByClass(id, { page: params.page ?? 1, limit: params.limit ?? 50 })
                    }
                    params={{}}
                    getRowId={(row) => row.id}
                    pageSize={10}
                    height={400}
                    columns={[
                        {
                            id: "student_name",
                            header: "Student",
                            cell: (row) => (
                                <Link
                                    href={`/students/${row.student_id}`}
                                    className="underline underline-offset-4 hover:no-underline"
                                >
                                    {row.student_name ?? "—"}
                                </Link>
                            ),
                            width: "1fr",
                        },
                        {
                            id: "admission_number",
                            header: "Admission",
                            cell: (row) => row.admission_number ?? "—",
                            width: "200px",
                        },
                        {
                            id: "status",
                            header: "Status",
                            cell: (row) => row.status ?? "—",
                            width: "120px",
                        },
                        {
                            id: "enrolled_at",
                            header: "Enrolled",
                            cell: (row) =>
                                row.enrolled_at
                                    ? new Date(row.enrolled_at).toLocaleDateString()
                                    : "—",
                            width: "120px",
                        },
                    ]}
                    emptyState={
                        <div className="text-muted-foreground p-4 text-center">
                            No enrolled students
                        </div>
                    }
                />
            </div>
        </div>
    );
}
