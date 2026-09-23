import { useMemo } from "react";
import Link from "next/link";
import { MoreVertical } from "lucide-react";
import { DataTable } from "@/components/shared/data-table";
import { listStudents, type StudentListItem } from "@/lib/api/students";
import { useDeleteStudents } from "../hooks/use-students";
import { Button } from "@/components/ui/button";
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
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
    const { mutateAsync: deleteStudents } = useDeleteStudents();

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
            deleteFn={deleteStudents}
            addHref="/students/add"
            pageSize={50}
            height={500}
        />
    );
}
