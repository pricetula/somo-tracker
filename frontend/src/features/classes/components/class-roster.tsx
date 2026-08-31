"use client";

import { DataTable } from "@/components/shared/data-table";
import { type DataTableColumn } from "@/components/shared/data-table/types";
import { getClassRoster, unenrollStudent, type RosterEntry } from "@/lib/api/classes";
import Link from "next/link";

interface ClassRosterProps {
    classId: string;
}
function buildColumns(classId: string): DataTableColumn<RosterEntry>[] {
    return [
        {
            id: "full_name",
            header: "Student Name",
            cell: (row) => <Link href={`/students/${row.id}`}>{row.full_name}</Link>,
        },
        {
            id: "admission_number",
            header: "Admission Number",
            width: "160px",
            cell: (row) => (
                <span className="text-muted-foreground">{row.admission_number || "-"}</span>
            ),
        },
        {
            id: "actions",
            header: "",
            width: "48px",
            align: "right",
            cell: (row) => <UnenrollCell classId={classId} student={row} />,
        },
    ];
}
function createRosterQueryFn(classId: string) {
    return (params: { page?: number; limit?: number; search?: string }) =>
        getClassRoster(classId, {
            page: params.page,
            limit: params.limit,
            search: params.search,
        });
}
function createBulkUnenrollFn(classId: string) {
    return async (id: string | number) => {
        await unenrollStudent(classId, String(id));
    };
}

import { UnenrollCell } from "./unenroll-cell";

export function ClassRoster({ classId }: ClassRosterProps) {
    const columns = buildColumns(classId);
    const rosterQueryFn = createRosterQueryFn(classId);
    const bulkUnenrollFn = createBulkUnenrollFn(classId);

    // Build addHref with academic term if available
    const addHref = `/classes/${classId}/enroll`;

    return (
        <DataTable
            isCheckable
            addHref={addHref}
            queryKey={["class-roster", classId]}
            queryFn={rosterQueryFn}
            columns={columns}
            getRowId={(row) => row.id}
            isSearchable
            searchPlaceholder="Search by student name or admission number..."
            deleteFn={bulkUnenrollFn}
            pageSize={13}
            height={480}
            emptyState="No students enrolled in this class yet."
            noResultsState="No students match your search."
        />
    );
}
