import { useMemo } from "react";
import Link from "next/link";
import { DataTable } from "@/components/shared/data-table";
import { listStudents, type StudentListItem } from "@/lib/api/students";

function listStudentsWithFilters(params: { page?: number; limit?: number; search?: string }) {
    return listStudents({
        page: params.page,
        limit: params.limit,
        search: params.search,
    });
}

export function StudentsTable() {
    const columns = useMemo(
        () => [
            {
                id: "admission_number",
                header: "Admission No.",
                cell: (row: StudentListItem) => row.admission_number,
                width: "1fr",
            },
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
                id: "dob",
                header: "Date of Birth",
                cell: (row: StudentListItem) => row.date_of_birth || "—",
                width: "1fr",
            },
            {
                id: "gender",
                header: "Gender",
                cell: (row: StudentListItem) => row.gender,
                width: "1fr",
            },
        ],
        []
    );

    return (
        <DataTable<
            StudentListItem,
            Record<string, never>,
            { items: StudentListItem[]; total: number }
        >
            queryKey={["students"]}
            queryFn={listStudentsWithFilters}
            getRowId={(row) => row.student_id}
            columns={columns}
            isCheckable
            isSearchable
            searchPlaceholder="Search by name or admission number…"
            addHref="/students/add"
            pageSize={50}
            height={600}
        />
    );
}
