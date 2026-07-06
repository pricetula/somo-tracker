"use client";

import { DataTable } from "@/components/shared/data-table";
import { listStudents } from "@/lib/api/students";
import type { Student } from "@/lib/api/students";

export function StudentsTable() {
    return (
        <DataTable
            queryKey={["students"]}
            queryFn={listStudents}
            getRowId={(row) => row.id}
            isSearchable
            searchPlaceholder="Search students..."
            pageSize={50}
            rowHeight={48}
            height={600}
            columns={[
                {
                    id: "name",
                    header: "Name",
                    width: "1fr",
                    cell: (row: Student) => (
                        <span className="text-sm font-medium">{row.full_name}</span>
                    ),
                },
                {
                    id: "gender",
                    header: "Gender",
                    width: "80px",
                    cell: (row: Student) => (
                        <span className="text-muted-foreground text-xs">{row.gender}</span>
                    ),
                },
                {
                    id: "class_name",
                    header: "Class",
                    width: "1fr",
                    cell: (row: Student) => (
                        <span className="text-muted-foreground text-xs">
                            {row.class_name ?? "—"}
                        </span>
                    ),
                },
                {
                    id: "is_active",
                    header: "Status",
                    width: "80px",
                    cell: (row: Student) => (
                        <span
                            className={
                                row.is_active
                                    ? "text-xs text-emerald-600"
                                    : "text-muted-foreground text-xs"
                            }
                        >
                            {row.is_active ? "Active" : "Inactive"}
                        </span>
                    ),
                },
            ]}
        />
    );
}
