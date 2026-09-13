import { useMemo } from "react";
import Link from "next/link";
import { DataTable } from "@/components/shared/data-table";
import { listAdmins, type AdminListItem } from "@/lib/api/admins";

function listAdminsWithFilters(params: {
    page?: number;
    limit?: number;
    search?: string;
    filters?: Record<string, string | string[]>;
    invitation_status?: string;
}) {
    const invitationStatus =
        typeof params.filters?.status === "string"
            ? params.filters.status
            : params.invitation_status;

    return listAdmins({
        page: params.page,
        limit: params.limit,
        search: params.search,
        invitation_status:
            invitationStatus === "all" || !invitationStatus
                ? undefined
                : (invitationStatus as "invited" | "accepted"),
    });
}

export function AdminsTable() {
    const filterGroups = useMemo(
        () => [
            {
                id: "invitation",
                label: "Invitation Status",
                items: [
                    {
                        id: "status",
                        label: "Status",
                        type: "sub_menu_single" as const,
                        submenu: [
                            { id: "all", label: "All", value: "all" },
                            { id: "invited", label: "Invited", value: "invited" },
                            { id: "accepted", label: "Accepted", value: "accepted" },
                        ],
                    },
                ],
            },
        ],
        []
    );

    const columns = useMemo(
        () => [
            {
                id: "name",
                header: "Name",
                cell: (row: AdminListItem) => (
                    <Link
                        href={`/admins/${row.membership_id}`}
                        className="underline underline-offset-4 hover:no-underline"
                    >
                        {row.full_name || "—"}
                    </Link>
                ),
                width: "2fr",
            },
            {
                id: "email",
                header: "Email",
                cell: (row: AdminListItem) => row.email,
                width: "2fr",
            },
            {
                id: "status",
                header: "Status",
                cell: (row: AdminListItem) => {
                    if (row.accepted_at) return "Accepted";
                    if (row.invited_at) return "Invited";
                    return "Pending";
                },
                width: "1fr",
            },
            {
                id: "active",
                header: "Active",
                cell: (row: AdminListItem) => (row.is_active ? "Yes" : "No"),
                width: "1fr",
            },
        ],
        []
    );

    return (
        <DataTable<
            AdminListItem,
            { filters?: Record<string, string | string[]> },
            { items: AdminListItem[]; total?: number }
        >
            queryKey={["admins"]}
            queryFn={listAdminsWithFilters}
            getRowId={(row) => row.membership_id}
            columns={columns}
            isSearchable
            searchPlaceholder="Search admins…"
            filterGroups={filterGroups}
            pageSize={50}
            height={600}
        />
    );
}
