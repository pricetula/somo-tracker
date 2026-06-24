/**
 * Invited Staff Table — displays non-accepted invitations by role.
 *
 * Shows pending, expired, revoked, and invite_failed invitations.
 * Each row shows full_name, full_name, email, status (badge),
 * expires_at, created_at with role-specific actions.
 */

"use client";

import * as React from "react";
import { useReactTable, getCoreRowModel, flexRender, type ColumnDef } from "@tanstack/react-table";
import { useVirtualizer } from "@tanstack/react-virtual";

import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import { MoreHorizontal } from "lucide-react";

import type { Invitation } from "@/lib/api/invitations";

// ─── Helpers ───────────────────────────────────────────────────────────────

type InvitationStatus = "pending" | "accepted" | "expired" | "revoked" | "invite_failed";

const STATUS_CONFIG: Record<
    InvitationStatus,
    { label: string; variant: "default" | "secondary" | "outline" | "destructive" }
> = {
    pending: { label: "Pending", variant: "secondary" },
    accepted: { label: "Accepted", variant: "default" },
    expired: { label: "Expired", variant: "outline" },
    revoked: { label: "Revoked", variant: "destructive" },
    invite_failed: { label: "Failed", variant: "destructive" },
};

function formatDate(iso: string): string {
    return new Date(iso).toLocaleDateString("en-US", {
        month: "short",
        day: "numeric",
        year: "numeric",
    });
}

function isExpired(expiresAt: string): boolean {
    return new Date(expiresAt) < new Date();
}

// ─── Columns ───────────────────────────────────────────────────────────────

function createColumns(): ColumnDef<Invitation>[] {
    return [
        {
            accessorKey: "full_name",
            header: "Full Name",
            cell: ({ row }) => {
                const name = row.original.full_name;
                return <span className="text-sm font-medium">{name || "—"}</span>;
            },
        },
        {
            accessorKey: "email",
            header: "Email",
            cell: ({ row }) => (
                <span className="text-muted-foreground text-sm">{row.original.email}</span>
            ),
        },
        {
            accessorKey: "status",
            header: "Status",
            cell: ({ row }) => {
                const status = row.original.status as InvitationStatus;
                const expired = status === "pending" && isExpired(row.original.expires_at);
                if (expired) {
                    return <Badge variant="outline">Expired</Badge>;
                }
                const config = STATUS_CONFIG[status] ?? {
                    label: status,
                    variant: "ghost" as const,
                };
                return <Badge variant={config.variant}>{config.label}</Badge>;
            },
        },
        {
            accessorKey: "expires_at",
            header: "Expires",
            cell: ({ row }) => {
                const expired = isExpired(row.original.expires_at);
                return (
                    <span
                        className={
                            expired
                                ? "text-destructive/70 text-sm"
                                : "text-muted-foreground text-sm"
                        }
                    >
                        {formatDate(row.original.expires_at)}
                    </span>
                );
            },
        },
        {
            accessorKey: "created_at",
            header: "Sent",
            cell: ({ row }) => (
                <span className="text-muted-foreground text-sm">
                    {formatDate(row.original.created_at)}
                </span>
            ),
        },
        {
            id: "actions",
            header: "",
            cell: ({ row }) => {
                const status = row.original.status as InvitationStatus;
                const expired = status === "pending" && isExpired(row.original.expires_at);

                return (
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
                            {/* pending → Resend, Revoke */}
                            {status === "pending" && !expired && (
                                <>
                                    {/* TODO: Implement Resend invitation */}
                                    <DropdownMenuItem>Resend</DropdownMenuItem>
                                    {/* TODO: Implement Revoke invitation */}
                                    <DropdownMenuItem>Revoke</DropdownMenuItem>
                                </>
                            )}
                            {/* expired → Resend */}
                            {(status === "expired" || expired) && (
                                <>
                                    {/* TODO: Implement Resend expired invitation */}
                                    <DropdownMenuItem>Resend</DropdownMenuItem>
                                </>
                            )}
                            {/* revoked → no actions */}
                            {status === "revoked" && (
                                <p className="text-muted-foreground px-2 py-1.5 text-xs">
                                    No actions
                                </p>
                            )}
                            {/* invite_failed → Fix & Retry */}
                            {status === "invite_failed" && (
                                <>
                                    {/* TODO: Implement recovery import grid re-hydration flow */}
                                    <DropdownMenuItem
                                        onClick={() =>
                                            console.log(
                                                "Fix & Retry for invitation",
                                                row.original.id
                                            )
                                        }
                                    >
                                        Fix & Retry
                                    </DropdownMenuItem>
                                </>
                            )}
                        </DropdownMenuContent>
                    </DropdownMenu>
                );
            },
        },
    ];
}

// ─── Props ─────────────────────────────────────────────────────────────────

interface InvitedStaffTableProps {
    invitations: Invitation[];
    total: number;
    roleLabel: string;
    isLoading: boolean;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function InvitedStaffTable({
    invitations,
    total,
    roleLabel,
    isLoading,
}: InvitedStaffTableProps) {
    const columns = React.useMemo(() => createColumns(), []);

    // eslint-disable-next-line react-hooks/incompatible-library
    const table = useReactTable({
        data: invitations,
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

    const skeletonRows = 10;

    return (
        <div className="flex flex-1 flex-col">
            {/* Table Canvas */}
            <div
                ref={parentRef}
                className="flex-1 overflow-auto"
                style={{ contain: "strict", minHeight: rows.length === 0 ? "200px" : undefined }}
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
                                            width:
                                                header.getSize() ||
                                                (header.id === "actions" ? 48 : "auto"),
                                            flex: header.id !== "actions" ? 1 : "0 0 auto",
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
                                    <Skeleton className="mr-3 h-4 w-20" />
                                    <Skeleton className="mr-3 h-4 w-20" />
                                    <Skeleton className="mr-3 h-4 w-36" />
                                    <Skeleton className="mr-3 h-4 w-16" />
                                    <Skeleton className="mr-3 h-4 w-24" />
                                    <Skeleton className="mr-3 h-4 w-24" />
                                </div>
                            ))
                        ) : rows.length === 0 ? (
                            <div className="flex items-center justify-center py-16">
                                <div className="text-center">
                                    <p className="text-muted-foreground text-sm font-medium">
                                        No pending invitations
                                    </p>
                                    <p className="text-muted-foreground mt-1 text-xs">
                                        All {roleLabel.toLowerCase()} invitations have been
                                        processed or accepted.
                                    </p>
                                </div>
                            </div>
                        ) : (
                            virtualizer.getVirtualItems().map((virtualRow) => {
                                const row = rows[virtualRow.index];
                                return (
                                    <div
                                        key={row.id}
                                        className="group border-border/40 hover:bg-muted/30 absolute right-0 left-0 flex h-12 items-center border-b transition-colors"
                                        style={{
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
                                                    width:
                                                        cell.column.getSize() ||
                                                        (cell.column.id === "actions"
                                                            ? 48
                                                            : "auto"),
                                                    flex:
                                                        cell.column.id !== "actions"
                                                            ? 1
                                                            : "0 0 auto",
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
                    {total} pending invitation{total !== 1 ? "s" : ""}
                </p>
            </div>
        </div>
    );
}
