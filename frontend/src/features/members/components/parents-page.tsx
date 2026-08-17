"use client";

import { DataTable } from "@/components/shared/data-table/data-table";
import { Button } from "@/components/ui/button";
import {
    listParents,
    deleteParent,
    type Parent,
    type ListParentsParams,
    type ListParentsResponse,
} from "@/lib/api/parents";
import { Trash2, Edit } from "lucide-react";

export function ParentsPage() {
    const columns = [
        { id: "email", header: "Email", cell: (row: Parent) => row.email },
        { id: "full_name", header: "Name", cell: (row: Parent) => row.full_name },
        { id: "role", header: "Role", cell: () => "Parent" },
        {
            id: "is_active",
            header: "Status",
            cell: (row: Parent) => (row.is_active ? "Active" : "Inactive"),
        },
        {
            id: "created_at",
            header: "Created At",
            cell: (row: Parent) => new Date(row.created_at).toLocaleDateString(),
        },
        {
            id: "actions",
            header: "Actions",
            cell: (row: Parent) => (
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
                                deleteParent(row.id).catch((err) =>
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
        <DataTable<Parent, ListParentsParams, ListParentsResponse>
            addHref="/parents/add"
            queryKey={["parents"]}
            queryFn={listParents}
            params={{}}
            columns={columns}
            getRowId={(row) => row.id}
            isSearchable
            searchPlaceholder="Search parents..."
            isCheckable
            deleteFn={(id) => deleteParent(String(id))}
        />
    );
}
