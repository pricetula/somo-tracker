"use client";

import { DataTable } from "@/components/shared/data-table";
import Link from "next/link";

type ClassRow = {
    id: string;
    grade: string; // natural sort key
    className: string;
    stream: string;
    students: number;
    teachers: number;
    attendance: "Marked" | "Not marked" | "Upcoming" | "No lessons";
};

const mockClasses: ClassRow[] = [
    {
        id: "c1",
        grade: "PP1",
        className: "PP1 Simba",
        stream: "Simba",
        students: 18,
        teachers: 4,
        attendance: "Marked",
    },
    {
        id: "c2",
        grade: "PP1",
        className: "PP1 Red house",
        stream: "Red house",
        students: 20,
        teachers: 4,
        attendance: "Not marked",
    },
    {
        id: "c3",
        grade: "PP2",
        className: "PP2 Simba",
        stream: "Simba",
        students: 22,
        teachers: 3,
        attendance: "Marked",
    },
    {
        id: "c4",
        grade: "PP2",
        className: "PP2 Red house",
        stream: "Red house",
        students: 20,
        teachers: 4,
        attendance: "Not marked",
    },
    {
        id: "c5",
        grade: "Grade 1",
        className: "Grade 1 Simba",
        stream: "Simba",
        students: 22,
        teachers: 0,
        attendance: "No lessons",
    },
    {
        id: "c6",
        grade: "Grade 1",
        className: "Grade 1 Red house",
        stream: "Red house",
        students: 24,
        teachers: 2,
        attendance: "Upcoming",
    },
    {
        id: "c7",
        grade: "Grade 6",
        className: "Grade 6 Simba",
        stream: "Simba",
        students: 18,
        teachers: 7,
        attendance: "Marked",
    },
    {
        id: "c8",
        grade: "Grade 6",
        className: "Grade 6 Red house",
        stream: "Red house",
        students: 24,
        teachers: 7,
        attendance: "Upcoming",
    },
];

// Natural grade order for sorting
const gradeOrder = [
    "PP1",
    "PP2",
    "Grade 1",
    "Grade 2",
    "Grade 3",
    "Grade 4",
    "Grade 5",
    "Grade 6",
    "Grade 7",
    "Grade 8",
    "Grade 9",
    "Grade 10",
    "Grade 11",
    "Grade 12",
];

const mockQueryFn = async () => {
    // Sort naturally by grade then by stream
    const sorted = [...mockClasses].sort((a, b) => {
        const ai = gradeOrder.indexOf(a.grade);
        const bi = gradeOrder.indexOf(b.grade);
        if (ai !== bi) return ai - bi;
        return a.stream.localeCompare(b.stream);
    });
    return { items: sorted, total: sorted.length };
};

export function ClassesTable() {
    const columns = [
        {
            id: "className",
            header: "Class",
            cell: (row: ClassRow) => (
                <Link
                    href={`/classes/${row.id}`}
                    className="text-foreground font-medium hover:underline"
                >
                    {row.className}
                </Link>
            ),
            width: "1.8fr",
        },
        {
            id: "stream",
            header: "Stream",
            cell: (row: ClassRow) => <span className="text-muted-foreground">{row.stream}</span>,
            width: "1fr",
        },
        {
            id: "students",
            header: "Students",
            cell: (row: ClassRow) => row.students,
            width: "0.8fr",
            align: "left" as const,
        },
        {
            id: "teachers",
            header: "Teachers",
            cell: (row: ClassRow) => {
                if (row.teachers === 0)
                    return <span className="text-muted-foreground">No lessons</span>;
                return `${row.teachers} teachers`;
            },
            width: "1fr",
        },
        {
            id: "attendance",
            header: "Attendance today",
            cell: (row: ClassRow) => {
                if (row.attendance === "No lessons")
                    return <span className="text-muted-foreground">—</span>;
                if (row.attendance === "Not marked")
                    return <span className="font-medium text-amber-600">Not marked</span>;
                if (row.attendance === "Marked")
                    return <span className="font-medium text-emerald-600">Marked</span>;
                return <span className="text-muted-foreground">Upcoming</span>;
            },
            width: "1.2fr",
        },
    ];

    const filterGroups = [
        {
            id: "stream",
            label: "Stream",
            items: [
                {
                    id: "stream",
                    label: "Stream",
                    type: "sub_menu_multi" as const,
                    submenu: [
                        { id: "simba", label: "Simba", value: "Simba" },
                        { id: "redhouse", label: "Red house", value: "Red house" },
                    ],
                },
            ],
        },
        {
            id: "grade",
            label: "Grade",
            items: [
                {
                    id: "grade",
                    label: "Grade",
                    type: "sub_menu_single" as const,
                    submenu: gradeOrder.map((g) => ({ id: g, label: g, value: g })),
                },
            ],
        },
    ];

    return (
        <div className="space-y-3">
            <div className="flex items-center justify-between gap-3">
                <h2 className="text-sm font-semibold">Classes</h2>
                <div className="flex gap-2">
                    {["Simba", "Red house"].map((s) => (
                        <span
                            key={s}
                            className="text-muted-foreground rounded-full border px-2.5 py-0.5 text-xs"
                        >
                            {s}
                        </span>
                    ))}
                </div>
            </div>
            <DataTable<ClassRow, Record<string, never>, { items: ClassRow[]; total: number }>
                queryKey={["dashboard-classes-mock"]}
                queryFn={mockQueryFn}
                columns={columns}
                getRowId={(row) => row.id}
                pageSize={10}
                height={320}
                isSearchable
                searchPlaceholder="Search class, stream, grade…"
                filterGroups={filterGroups}
                addHref="/classes/add"
            />
            <div className="text-muted-foreground text-xs">
                Showing 5 of 28 ·{" "}
                <Link href="/classes" className="underline">
                    View all classes →
                </Link>
            </div>
        </div>
    );
}
