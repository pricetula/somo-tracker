"use client";

import { DataTable } from "@/components/shared/data-table/data-table";
import { Button } from "@/components/ui/button";
import {
    listAdmins,
    deleteAdmin,
    type Member,
    type ListAdminsParams,
    type ListMembersResponse,
} from "@/lib/api/admins";
import { Trash2, Edit } from "lucide-react";
import { useRouter } from "next/navigation";

export default function AdminsPage() {
    const router = useRouter();

    const columns = [
        { id: "email", header: "Email", cell: (row: Member) => row.email },
        { id: "full_name", header: "Name", cell: (row: Member) => row.full_name },
        { id: "role", header: "Role", cell: (row: Member) => row.role },
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
                                deleteAdmin(row.id).catch((err) =>
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
                <h1 className="text-xl font-semibold">Admins</h1>
                <Button variant="outline" onClick={() => router.push("/admins/bulk-invite")}>
                    Bulk Invite
                </Button>
            </div>

            <DataTable<Member, ListAdminsParams, ListMembersResponse>
                queryKey={["admins"]}
                queryFn={listAdmins}
                params={{}}
                columns={columns}
                getRowId={(row) => row.id}
                isSearchable
                searchPlaceholder="Search admins..."
                isCheckable
                deleteFn={(id) => deleteAdmin(String(id))}
            />
        </div>
    );
}
