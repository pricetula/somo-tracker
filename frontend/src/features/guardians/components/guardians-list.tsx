import { useMemo, useCallback } from "react";
import Link from "next/link";
import { MoreVertical } from "lucide-react";
import { DataTable } from "@/components/shared/data-table";
import { listGuardians, type GuardianListItem } from "@/lib/api/guardians";
import { useDeleteGuardians } from "../hooks/use-guardians";
import { Button } from "@/components/ui/button";
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

function listGuardiansWithFilters(params: {
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

    return listGuardians({
        page: params.page,
        limit: params.limit,
        search: params.search,
        invitation_status:
            invitationStatus === "all" || !invitationStatus
                ? undefined
                : (invitationStatus as "invited" | "accepted"),
    });
}

export function GuardiansTable() {
    const { mutateAsync: deleteGuardians } = useDeleteGuardians();
    const handleDelete = useCallback(
        async (id: string | number) => {
            await deleteGuardians([String(id)]);
        },
        [deleteGuardians]
    );
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
                cell: (row: GuardianListItem) => (
                    <Link
                        href={`/guardians/${row.membership_id}`}
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
                cell: (row: GuardianListItem) => row.email,
                width: "2fr",
            },
            {
                id: "status",
                header: "Status",
                cell: (row: GuardianListItem) => {
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
                cell: (row: GuardianListItem) => (
                    <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                            <Button variant="ghost" size="icon">
                                <MoreVertical className="size-4" />
                            </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                            <DropdownMenuItem asChild>
                                <Link href={`/guardians/${row.membership_id}`}>Edit</Link>
                            </DropdownMenuItem>
                            <DropdownMenuItem
                                onSelect={() => {
                                    void handleDelete(row.user_id);
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
            GuardianListItem,
            { filters?: Record<string, string | string[]> },
            { items: GuardianListItem[]; total?: number }
        >
            queryKey={["guardians"]}
            queryFn={listGuardiansWithFilters}
            getRowId={(row) => row.user_id}
            columns={columns}
            isCheckable
            isSearchable
            searchPlaceholder="Search guardians…"
            filterGroups={filterGroups}
            deleteFn={handleDelete}
            pageSize={50}
            height={600}
        />
    );
}
