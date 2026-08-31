"use client";

import { DataTable } from "@/components/shared/data-table/data-table";
import { Button } from "@/components/ui/button";
import {
    listStudents,
    deleteStudent,
    type Student,
    type ListStudentsParams,
    type ListStudentsResponse,
} from "@/lib/api/students";
import { Trash2, Edit } from "lucide-react";

export function StudentsPage() {
    const columns = [
        { id: "full_name", header: "Full Name", cell: (row: Student) => row.full_name },
        { id: "gender", header: "Gender", cell: (row: Student) => row.gender ?? "-" },
        {
            id: "date_of_birth",
            header: "Date of Birth",
            cell: (row: Student) => row.date_of_birth ?? "-",
        },
        { id: "class_name", header: "Class", cell: (row: Student) => row.class_name ?? "-" },
        {
            id: "is_active",
            header: "Status",
            cell: (row: Student) => (row.is_active ? "Active" : "Inactive"),
        },
        {
            id: "created_at",
            header: "Created At",
            cell: (row: Student) => new Date(row.created_at).toLocaleDateString(),
        },
        {
            id: "actions",
            header: "Actions",
            cell: (row: Student) => (
                <div className="flex items-center gap-2">
                    <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => alert(`Edit ${row.full_name}`)}
                    >
                        <Edit className="size-3" />
                    </Button>
                    <Button
                        variant="destructive"
                        size="icon"
                        onClick={() => {
                            if (window.confirm(`Delete ${row.full_name}?`)) {
                                deleteStudent(row.id).catch((err) =>
                                    alert(`Failed to delete: ${err}`)
                                );
                            }
                        }}
                    >
                        <Trash2 className="size-3" />
                    </Button>
                </div>
            ),
        },
    ];

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <h1 className="text-xl font-semibold">Students</h1>
                <Button
                    variant="outline"
                    onClick={() =>
                        alert(
                            "Bulk invite for students is not available. Use student import instead."
                        )
                    }
                >
                    Bulk Invite
                </Button>
            </div>

            <DataTable<Student, ListStudentsParams, ListStudentsResponse>
                queryKey={["students"]}
                queryFn={listStudents}
                params={{}}
                columns={columns}
                getRowId={(row: Student) => row.id}
                isSearchable
                searchPlaceholder="Search students..."
                isCheckable
                deleteFn={(id) => deleteStudent(String(id))}
            />
        </div>
    );
}
