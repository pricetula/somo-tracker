/**
 * Admins Listing Table — displays school admin users.
 *
 * Columns: Full Name, Email, Account Status (toggle), Actions (delete).
 * Search targets: User Name or Email Address.
 * Bulk Import opens the intercepted parallel route modal.
 *
 * Uses TanStack Table + TanStack Virtual for performance.
 */

"use client";

import * as React from "react";
import Link from "next/link";
import { useReactTable, getCoreRowModel, flexRender, type ColumnDef } from "@tanstack/react-table";
import { useVirtualizer } from "@tanstack/react-virtual";

import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Input } from "@/components/ui/input";
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { MoreHorizontal, Search, Upload } from "lucide-react";

import { StatusToggleCell } from "./status-toggle-cell";
import type { Member } from "@/lib/api/admins";
import { useToggleAdminActive, useDeleteAdmin } from "../hooks/use-admins";

// ─── Columns ───────────────────────────────────────────────────────────────

function createColumns(
    toggleMutation: ReturnType<typeof useToggleAdminActive>,
    onDeleteRequest: (member: Member) => void
): ColumnDef<Member>[] {
    return [
        {
            accessorKey: "full_name",
            header: "Full Name",
            cell: ({ row }) => (
                <span className="text-sm font-medium">{row.original.full_name || "—"}</span>
            ),
        },
        {
            accessorKey: "email",
            header: "Email",
            cell: ({ row }) => (
                <span className="text-muted-foreground text-sm">{row.original.email}</span>
            ),
        },
        {
            id: "is_active",
            header: "Account Status",
            cell: ({ row }) => (
                <StatusToggleCell
                    member={row.original}
                    onToggle={(userId, isActive) => toggleMutation.mutate({ userId, isActive })}
                    isPending={toggleMutation.isPending}
                    label={{
                        activate: "Activate admin",
                        deactivate: "Deactivate admin",
                    }}
                />
            ),
        },
        {
            id: "actions",
            header: "",
            cell: ({ row }) => (
                <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                        <Button
                            variant="ghost"
                            size="icon-sm"
                            className="opacity-0 transition-opacity group-hover:opacity-100"
                        >
                            <MoreHorizontal className="size-4" />
                            <span className="sr-only">Actions</span>
                        </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end" className="w-36">
                        <DropdownMenuItem
                            className="text-destructive focus:text-destructive"
                            onClick={() => onDeleteRequest(row.original)}
                        >
                            Delete
                        </DropdownMenuItem>
                    </DropdownMenuContent>
                </DropdownMenu>
            ),
        },
    ];
}

// ─── Props ─────────────────────────────────────────────────────────────────

interface AdminsTableProps {
    admins: Member[];
    total: number;
    isLoading: boolean;
    search: string;
    onSearchChange: (value: string) => void;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function AdminsTable({
    admins,
    total,
    isLoading,
    search,
    onSearchChange,
}: AdminsTableProps) {
    const toggleMutation = useToggleAdminActive();
    const deleteMutation = useDeleteAdmin();
    const [memberToDelete, setMemberToDelete] = React.useState<Member | null>(null);

    const columns = React.useMemo(
        () => createColumns(toggleMutation, setMemberToDelete),
        [toggleMutation]
    );

    // eslint-disable-next-line react-hooks/incompatible-library
    const table = useReactTable({
        data: admins,
        columns,
        getCoreRowModel: getCoreRowModel(),
    });

    const parentRef = React.useRef<HTMLDivElement>(null);
    const rows = table.getRowModel().rows;

    const virtualizer = useVirtualizer({
        count: rows.length,
        getScrollElement: () => parentRef.current,
        estimateSize: () => 48,
        overscan: 10,
    });

    const skeletonRows = 8;

    const handleDeleteConfirm = () => {
        if (!memberToDelete) return;
        deleteMutation.mutate(memberToDelete.id, {
            onSettled: () => setMemberToDelete(null),
        });
    };

    return (
        <div className="flex flex-1 flex-col">
            {/* Search bar */}
            <div className="mb-3 flex items-center gap-3">
                <div className="relative max-w-xs flex-1">
                    <Search className="text-muted-foreground absolute top-2.5 left-2.5 size-4" />
                    <Input
                        placeholder="Search by name or email…"
                        value={search}
                        onChange={(e) => onSearchChange(e.target.value)}
                        className="pl-8"
                    />
                </div>
                <Button variant="outline" size="sm" asChild>
                    <Link href="/admins/import">
                        <Upload className="mr-1.5 size-3.5" />
                        Bulk Import
                    </Link>
                </Button>
            </div>

            {/* Table Canvas */}
            <div
                ref={parentRef}
                className="flex-1 overflow-auto"
                style={{
                    contain: "layout paint",
                    minHeight: rows.length === 0 ? "200px" : undefined,
                }}
            >
                <div className="min-w-175">
                    {/* Sticky Header */}
                    <div className="bg-background/95 sticky top-0 z-10 rounded-lg backdrop-blur-sm">
                        {table.getHeaderGroups().map((headerGroup) => (
                            <div key={headerGroup.id} className="border-border/40 flex border-b">
                                {headerGroup.headers.map((header) => (
                                    <div
                                        key={header.id}
                                        className="text-muted-foreground flex h-10 items-center px-3 text-xs font-medium tracking-wider uppercase"
                                        style={{
                                            flex: header.id !== "actions" ? 1 : "0 0 auto",
                                            width: header.id === "actions" ? 48 : "auto",
                                        }}
                                    >
                                        {flexRender(
                                            header.column.columnDef.header,
                                            header.getContext()
                                        )}
                                    </div>
                                ))}
                            </div>
                        ))}
                    </div>

                    {/* Virtualized Body */}
                    <div
                        style={{
                            height: `${virtualizer.getTotalSize()}px`,
                            position: "relative",
                        }}
                    >
                        {isLoading && rows.length === 0 ? (
                            Array.from({ length: skeletonRows }).map((_, i) => (
                                <div
                                    key={`skeleton-${i}`}
                                    className="border-border/40 flex h-12 items-center border-b px-3"
                                >
                                    <Skeleton className="mr-3 h-4 w-20 flex-1" />
                                    <Skeleton className="mr-3 h-4 w-20 flex-1" />
                                    <Skeleton className="mr-3 h-6 w-16 flex-1" />
                                    <Skeleton className="mr-3 h-4 w-4 flex-1" />
                                </div>
                            ))
                        ) : rows.length === 0 ? (
                            <div className="flex items-center justify-center py-16">
                                <div className="text-center">
                                    <p className="text-muted-foreground text-sm font-medium">
                                        {search ? "No admins match your search" : "No admins yet"}
                                    </p>
                                    <p className="text-muted-foreground mt-1 text-xs">
                                        {search
                                            ? "Try a different search term."
                                            : "Invite admins to manage your school."}
                                    </p>
                                </div>
                            </div>
                        ) : (
                            virtualizer.getVirtualItems().map((virtualRow) => {
                                const row = rows[virtualRow.index];
                                return (
                                    <div
                                        key={virtualRow.key}
                                        className="group border-border/40 hover:bg-muted/30 absolute right-0 left-0 flex items-center border-b transition-colors"
                                        style={{
                                            position: "absolute",
                                            top: 0,
                                            left: 0,
                                            width: "100%",
                                            height: `${virtualRow.size}px`,
                                            transform: `translateY(${virtualRow.start}px)`,
                                        }}
                                    >
                                        {row.getVisibleCells().map((cell) => (
                                            <div
                                                key={cell.id}
                                                className={
                                                    "flex items-center truncate px-3 text-sm" +
                                                    (cell.column.id === "actions"
                                                        ? " justify-end"
                                                        : "")
                                                }
                                                style={{
                                                    flex:
                                                        cell.column.id !== "actions"
                                                            ? 1
                                                            : "0 0 auto",
                                                    width:
                                                        cell.column.id === "actions" ? 48 : "auto",
                                                }}
                                            >
                                                {flexRender(
                                                    cell.column.columnDef.cell,
                                                    cell.getContext()
                                                )}
                                            </div>
                                        ))}
                                    </div>
                                );
                            })
                        )}
                    </div>
                </div>
            </div>

            {/* Footer counter */}
            <div className="border-border/40 flex items-center justify-between border-t px-3 py-2">
                <p className="text-muted-foreground text-xs">
                    {total} admin{total !== 1 ? "s" : ""}
                </p>
            </div>

            {/* Delete confirmation dialog */}
            <AlertDialog
                open={!!memberToDelete}
                onOpenChange={(open) => {
                    if (!open) setMemberToDelete(null);
                }}
            >
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>Delete admin</AlertDialogTitle>
                        <AlertDialogDescription>
                            Are you sure you want to permanently delete{" "}
                            <strong>{memberToDelete?.full_name || memberToDelete?.email}</strong>?
                            This action cannot be undone.
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel disabled={deleteMutation.isPending}>
                            Cancel
                        </AlertDialogCancel>
                        <AlertDialogAction
                            onClick={handleDeleteConfirm}
                            disabled={deleteMutation.isPending}
                        >
                            {deleteMutation.isPending ? "Deleting..." : "Delete"}
                        </AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </div>
    );
}
