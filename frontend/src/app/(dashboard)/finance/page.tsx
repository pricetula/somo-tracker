"use client";

import { DataTable } from "@/components/shared/data-table/data-table";
import { Button } from "@/components/ui/button";
import {
    listFinanceStaff,
    deleteFinanceStaff,
    type Member,
    type ListFinanceStaffParams,
    type ListMembersResponse,
} from "@/lib/api/finance";
import { Trash2, Edit } from "lucide-react";

export default function FinancePage() {
    const columns = [
        { id: "email", header: "Email", cell: (row: Member) => row.email },
        { id: "full_name", header: "Name", cell: (row: Member) => row.full_name },
        { id: "role", header: "Role", cell: () => "Finance Staff" },
        {
            id: "is_active",
            header: "Status",
            cell: (row: Member) => (row.is_active ? "Active" : "Inactive"),
        },
        {
            id: "created_at",
            header: "Created At",
            cell: (row: Member) => new Date(row.created_at).toLocaleDateString(),
        },
        {
            id: "actions",
            header: "Actions",
            cell: (row: Member) => (
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
                                deleteFinanceStaff(row.id).catch((err) =>
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
        <DataTable<Member, ListFinanceStaffParams, ListMembersResponse>
            addHref="/finance/add"
            queryKey={["finance"]}
            queryFn={listFinanceStaff}
            params={{}}
            columns={columns}
            getRowId={(row) => row.id}
            isSearchable
            searchPlaceholder="Search finance staff..."
            isCheckable
            deleteFn={(id) => deleteFinanceStaff(String(id))}
        />
    );
}
