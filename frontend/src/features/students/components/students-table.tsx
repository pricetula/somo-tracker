"use client";

import { useMemo, useState } from "react";
import { DataTable } from "@/components/shared/data-table";
import { listStudents } from "@/lib/api/students";
import type { Student } from "@/lib/api/students";

export function StudentsTable() {
    // Any filter that should trigger a refetch (search included) lives here,
    // and gets passed straight through as `params` to listClasses.
    const [search, setSearch] = useState("");

    // Memoize to keep the object reference stable — otherwise DataTable's
    // infinite query key changes every render, keeping isPending stuck true.
    const params = useMemo(() => ({ search: search || undefined }), [search]);

    return (
        <DataTable
            queryKey={["students", search]}
            queryFn={listStudents}
            params={params}
            getRowId={(row) => row.id}
            onSearch={setSearch}
            searchPlaceholder="Search students..."
            pageSize={50}
            rowHeight={48}
            height={600}
            columns={[
                {
                    id: "name",
                    header: "Name",
                    width: "2fr",
                    cell: (row: Student) => row.full_name,
                },
            ]}
        />
    );
}
