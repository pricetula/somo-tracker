"use client";

import { DataTable } from "@/components/shared/data-table/data-table";
import { Button } from "@/components/ui/button";
import {
    listNurses,
    deleteNurse,
    type Member,
    type ListNursesParams,
    type ListMembersResponse,
} from "@/lib/api/nurses";
import { Trash2, Edit } from "lucide-react";
import { useRouter } from "next/navigation";

export default function NursesPage() {
    const router = useRouter();

    const columns = [
        { id: "email", header: "Email", cell: (row: Member) => row.email },
        { id: "full_name", header: "Name", cell: (row: Member) => row.full_name },
        { id: "role", header: "Role", cell: () => "Nurse" },
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
                                deleteNurse(row.id).catch((err) =>
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
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <h1 className="text-xl font-semibold">Nurses</h1>
                <Button variant="outline" onClick={() => router.push("/nurses/bulk-invite")}>
                    Bulk Invite
                </Button>
            </div>

            <DataTable<Member, ListNursesParams, ListMembersResponse>
                queryKey={["nurses"]}
                queryFn={listNurses}
                params={{}}
                columns={columns}
                getRowId={(row) => row.id}
                isSearchable
                searchPlaceholder="Search nurses..."
                isCheckable
                deleteFn={(id) => deleteNurse(String(id))}
            />
        </div>
    );
}
