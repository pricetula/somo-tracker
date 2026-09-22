import { useMemo } from "react";
import Link from "next/link";
import { DataTable } from "@/components/shared/data-table";
import { listStudents, type StudentListItem } from "@/lib/api/students";
import { studentFilterGroups, mapStudentFiltersToParams } from "./students-filters";

function listStudentsWithFilters(params: {
    page?: number;
    limit?: number;
    search?: string;
    filters?: Record<string, string | string[]>;
}) {
    const { class_id } = mapStudentFiltersToParams(params.filters);
    return listStudents({
        page: params.page,
        limit: params.limit,
        search: params.search,
        class_id,
    });
}

export function StudentsTable() {
    const filterGroups = useMemo(() => studentFilterGroups, []);
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
                id: "class",
                header: "Class",
                cell: (row: StudentListItem) => row.class_name || "—",
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
            addHref="/students/add"
            pageSize={50}
            height={600}
        />
    );
}
