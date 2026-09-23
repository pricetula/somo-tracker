import { useMemo, useCallback } from "react";
import Link from "next/link";
import { MoreVertical } from "lucide-react";
import { DataTable } from "@/components/shared/data-table";
import { listAdmins, type AdminListItem } from "@/lib/api/admins";
import { useDeleteAdmins } from "../hooks/use-admins";
import { Button } from "@/components/ui/button";
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { adminFilterGroups, mapAdminFiltersToParams } from "./admins-filters";

function listAdminsWithFilters(params: {
    page?: number;
    limit?: number;
    search?: string;
    filters?: Record<string, string | string[]>;
    invitation_status?: string;
}) {
    const { invitation_status } = mapAdminFiltersToParams(params.filters);
    return listAdmins({
        page: params.page,
        limit: params.limit,
        search: params.search,
        invitation_status: invitation_status ?? params.invitation_status,
    });
}

export function AdminsTable() {
    const { mutateAsync: deleteAdmins } = useDeleteAdmins();
    const handleDelete = useCallback(
        async (ids: string[]) => {
            await deleteAdmins(ids);
        },
        [deleteAdmins]
    );
    const filterGroups = useMemo(() => adminFilterGroups, []);

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
                    if (row.accepted_at) return "Accepted Invite";
                    if (row.invited_at) return "Invited";
                    if (row.is_active) return "Active";
                    return "Pending";
                },
                width: "1fr",
            },
            {
                id: "actions",
                header: "",
                cell: (row: AdminListItem) => (
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
                                render={<Link href={`/admins/${row.membership_id}`}>Edit</Link>}
                            />
                            <DropdownMenuItem
                                onSelect={() => {
                                    void handleDelete([row.user_id]);
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
        [handleDelete]
    );

    return (
        <DataTable<
            AdminListItem,
            { filters?: Record<string, string | string[]> },
            { items: AdminListItem[]; total?: number }
        >
            queryKey={["admins"]}
            queryFn={listAdminsWithFilters}
            getRowId={(row) => row.user_id}
            columns={columns}
            isCheckable
            isSearchable
            searchPlaceholder="Search admins…"
            filterGroups={filterGroups}
            deleteFn={handleDelete}
            addHref="/admins/invite"
            pageSize={50}
            height={500}
        />
    );
}
