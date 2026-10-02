"use client";
import { useMemo, useCallback } from "react";
import Link from "next/link";
import { DataTable } from "@/components/shared/data-table";
import { listRooms } from "../services/api";
import type { Room, ListRoomsParams } from "../types/room";
import { useDeleteRoom } from "../hooks/use-rooms";
import { MoreVertical } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

function listWithFilters(params: ListRoomsParams) {
    return listRooms(params);
}

export function RoomsTable() {
    const { mutateAsync: deleteRoom } = useDeleteRoom();
    const handleDelete = useCallback(
        async (ids: string[]) => {
            await deleteRoom(ids[0]);
        },
        [deleteRoom]
    );

    const filterGroups = useMemo(
        () => [
            {
                id: "roomType",
                label: "Room Type",
                items: [
                    {
                        id: "roomType-filter",
                        label: "Room Type",
                        type: "sub_menu_multi" as const,
                        submenu: [
                            { id: "STANDARD", label: "Standard", value: "STANDARD" },
                            { id: "SCIENCE_LAB", label: "Science Lab", value: "SCIENCE_LAB" },
                            { id: "COMPUTER_LAB", label: "Computer Lab", value: "COMPUTER_LAB" },
                            { id: "GYM", label: "Gym", value: "GYM" },
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
                cell: (row: Room) => (
                    <Link
                        href={`/school/rooms/${row.id}`}
                        className="underline underline-offset-4 hover:no-underline"
                    >
                        {row.name}
                    </Link>
                ),
                width: "2fr",
            },
            {
                id: "roomType",
                header: "Type",
                cell: (row: Room) => (
                    <span className="bg-muted text-muted-foreground inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium">
                        {row.roomType.replace("_", " ")}
                    </span>
                ),
                width: "1.5fr",
            },
            {
                id: "capacity",
                header: "Capacity",
                cell: (row: Room) => row.capacity?.toString() ?? "—",
                width: "1fr",
                align: "center" as const,
            },
            {
                id: "actions",
                header: "",
                cell: (row: Room) => (
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
                                render={<Link href={`/school/rooms/${row.id}`}>Edit</Link>}
                            />
                            <DropdownMenuItem
                                onSelect={() => handleDelete([row.id])}
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
            Room,
            ListRoomsParams,
            { items: Room[]; total: number; page: number; limit: number }
        >
            queryKey={["rooms", "list"]}
            queryFn={listWithFilters}
            getRowId={(row) => row.id}
            columns={columns}
            isSearchable
            searchPlaceholder="Search rooms…"
            filterGroups={filterGroups}
            addHref="/school/rooms/add"
            pageSize={50}
            height={500}
            deleteFn={async (ids) => {
                await deleteRoom(ids[0]);
            }}
        />
    );
}
