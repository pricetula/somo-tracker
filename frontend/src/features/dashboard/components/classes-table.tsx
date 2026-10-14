"use client";

import { DataTable } from "@/components/shared/data-table";

type ClassRow = {
    id: string;
    className: string;
    subjectTeacher: string;
    attendance: "marked" | "not marked" | "upcoming";
};

const mockClasses: ClassRow[] = [
    { id: "1", className: "PP1 Simba", subjectTeacher: "Mary Achieng", attendance: "marked" },
    { id: "2", className: "PP1 Red house", subjectTeacher: "Joseph Kamau", attendance: "marked" },
    {
        id: "3",
        className: "PP2 Red house",
        subjectTeacher: "Grace Wanjiru",
        attendance: "not marked",
    },
    { id: "4", className: "Grade 1 Simba", subjectTeacher: "Unassigned", attendance: "not marked" },
    { id: "5", className: "Grade 2 Simba", subjectTeacher: "Peter Otieno", attendance: "marked" },
    { id: "6", className: "Grade 2 Red house", subjectTeacher: "Lucy Njeri", attendance: "marked" },
    {
        id: "7",
        className: "Grade 3 Simba",
        subjectTeacher: "Achieng Otieno",
        attendance: "upcoming",
    },
    {
        id: "8",
        className: "Grade 4 Red house",
        subjectTeacher: "Wanjiru Kimani",
        attendance: "marked",
    },
];

const mockQueryFn = async () => {
    return { items: mockClasses, total: mockClasses.length };
};

export function ClassesTable() {
    const columns = [
        {
            id: "className",
            header: "Class name",
            cell: (row: ClassRow) => row.className,
            width: "2fr",
        },
        {
            id: "subjectTeacher",
            header: "Subject teacher",
            cell: (row: ClassRow) => row.subjectTeacher,
            width: "2fr",
        },
        {
            id: "attendance",
            header: "Attendance today",
            cell: (row: ClassRow) => {
                const label = row.attendance;
                const color =
                    label === "marked"
                        ? "text-emerald-600"
                        : label === "not marked"
                          ? "text-amber-600"
                          : "text-muted-foreground";
                return <span className={color}>{label}</span>;
            },
            width: "1.5fr",
        },
    ];

    return (
        <div className="space-y-3">
            <h2 className="text-foreground text-sm font-medium">CLASSES</h2>
            <DataTable<ClassRow, Record<string, never>, { items: ClassRow[]; total: number }>
                queryKey={["dashboard-classes-mock"]}
                queryFn={mockQueryFn}
                columns={columns}
                getRowId={(row) => row.id}
                pageSize={10}
                height={360}
                isSearchable
                searchPlaceholder="Search class…"
            />
        </div>
    );
}
