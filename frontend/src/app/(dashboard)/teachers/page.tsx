"use client";

import { DataTable } from "@/components/shared/data-table/data-table";
import { Button } from "@/components/ui/button";
import {
    listTeachers,
    deleteTeacher,
    type TeacherMember,
    type ListTeachersParams,
    type ListTeachersResponse,
} from "@/lib/api/teachers";
import { Trash2, Edit } from "lucide-react";
import { useRouter } from "next/navigation";

export default function TeachersPage() {
    const router = useRouter();

    const columns = [
        { id: "email", header: "Email", cell: (row: TeacherMember) => row.email },
        { id: "full_name", header: "Name", cell: (row: TeacherMember) => row.full_name },
        { id: "role", header: "Role", cell: () => "Teacher" },
        {
            id: "tsc_number",
            header: "TSC Number",
            cell: (row: TeacherMember) => row.tsc_number ?? "-",
        },
        {
            id: "knec_panel_assessor_id",
            header: "KNEC Panel Assessor",
            cell: (row: TeacherMember) => row.knec_panel_assessor_id ?? "-",
        },
        {
            id: "teacher_role",
            header: "Teacher Role",
            cell: (row: TeacherMember) => row.teacher_role ?? "-",
        },
        {
            id: "is_active",
            header: "Status",
            cell: (row: TeacherMember) => (row.is_active ? "Active" : "Inactive"),
        },
        {
            id: "created_at",
            header: "Created At",
            cell: (row: TeacherMember) => new Date(row.created_at).toLocaleDateString(),
        },
        {
            id: "actions",
            header: "Actions",
            cell: (row: TeacherMember) => (
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
                                deleteTeacher(row.id).catch((err) =>
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
                <h1 className="text-xl font-semibold">Teachers</h1>
                <Button variant="outline" onClick={() => router.push("/teachers/bulk-invite")}>
                    Bulk Invite
                </Button>
            </div>

            <DataTable<TeacherMember, ListTeachersParams, ListTeachersResponse>
                queryKey={["teachers"]}
                queryFn={listTeachers}
                params={{}}
                columns={columns}
                getRowId={(row) => row.id}
                isSearchable
                searchPlaceholder="Search teachers..."
                isCheckable
                deleteFn={(id) => deleteTeacher(String(id))}
            />
        </div>
    );
}
