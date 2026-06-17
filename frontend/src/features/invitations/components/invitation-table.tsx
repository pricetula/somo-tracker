/**
 * Invitation Table — virtual table with search, email, status, role, and
 * expired filters. Follows the same pattern as the member table.
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
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { MoreHorizontal, Search, Mail, X } from "lucide-react";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";

import type { Invitation, InvitationStatus, InvitationRole } from "@/lib/api/invitations";

// ─── Helpers ───────────────────────────────────────────────────────────────

const STATUS_LABELS: Record<
    InvitationStatus,
    { label: string; variant: "default" | "secondary" | "outline" | "ghost" | "destructive" }
> = {
    pending: { label: "Pending", variant: "secondary" },
    accepted: { label: "Accepted", variant: "default" },
    expired: { label: "Expired", variant: "outline" },
    revoked: { label: "Revoked", variant: "destructive" },
};

const ROLE_LABELS: Record<InvitationRole, string> = {
    SYSTEM_ADMIN: "System Admin",
    SCHOOL_ADMIN: "School Admin",
    TEACHER: "Teacher",
    SUPPORT_STAFF: "Staff",
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

const columns: ColumnDef<Invitation>[] = [
    {
        accessorKey: "first_name",
        header: "Name",
        cell: ({ row }) => {
            const first = row.original.first_name;
            const last = row.original.last_name;
            const name = [first, last].filter(Boolean).join(" ");
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
        accessorKey: "role",
        header: "Role",
        cell: ({ row }) => (
            <span className="text-muted-foreground text-sm capitalize">
                {ROLE_LABELS[row.original.role] ||
                    row.original.role.toLowerCase().replace(/_/g, " ")}
            </span>
        ),
    },
    {
        accessorKey: "status",
        header: "Status",
        cell: ({ row }) => {
            const status = row.original.status as InvitationStatus;
            const info = STATUS_LABELS[status] ?? { label: status, variant: "ghost" as const };
            const expired = status === "pending" && isExpired(row.original.expires_at);
            return (
                <Badge variant={expired ? "outline" : info.variant}>
                    {expired ? "Expired" : info.label}
                </Badge>
            );
        },
    },
    {
        accessorKey: "expires_at",
        header: "Expires",
        cell: ({ row }) => {
            const expired = isExpired(row.original.expires_at);
            return (
                <span
                    className={cn(
                        "text-muted-foreground text-sm",
                        expired && "text-destructive/70"
                    )}
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
        cell: () => (
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
                    <DropdownMenuItem>Resend Invitation</DropdownMenuItem>
                    <DropdownMenuItem>Revoke</DropdownMenuItem>
                </DropdownMenuContent>
            </DropdownMenu>
        ),
    },
];

// ─── Props ─────────────────────────────────────────────────────────────────

interface InvitationTableProps {
    invitations: Invitation[];
    total: number;
    search: string;
    onSearchChange: (value: string) => void;
    emailFilter: string;
    onEmailFilterChange: (value: string) => void;
    statusFilter: InvitationStatus | "";
    onStatusFilterChange: (value: InvitationStatus | "") => void;
    roleFilter: InvitationRole | "";
    onRoleFilterChange: (value: InvitationRole | "") => void;
    showExpired: boolean;
    onShowExpiredChange: (value: boolean) => void;
    onInviteClick: () => void;
    isLoading: boolean;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function InvitationTable({
    invitations,
    total,
    search,
    onSearchChange,
    emailFilter,
    onEmailFilterChange,
    statusFilter,
    onStatusFilterChange,
    roleFilter,
    onRoleFilterChange,
    showExpired,
    onShowExpiredChange,
    onInviteClick,
    isLoading,
}: InvitationTableProps) {
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
    const hasActiveFilters = search || emailFilter || statusFilter || roleFilter || !showExpired;

    return (
        <div className="flex flex-1 flex-col">
            {/* Controls Row */}
            <div className="flex flex-wrap items-center gap-3 px-6 py-3">
                {/* Name search */}
                <div className="relative max-w-xs flex-1">
                    <Search className="text-muted-foreground pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2" />
                    <Input
                        placeholder="Search by name..."
                        value={search}
                        onChange={(e) => onSearchChange(e.target.value)}
                        className="bg-muted/50 h-9 border-none pl-8 text-sm"
                    />
                </div>

                {/* Email filter */}
                <div className="relative max-w-[200px] flex-1">
                    <Mail className="text-muted-foreground pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2" />
                    <Input
                        placeholder="Filter by email..."
                        value={emailFilter}
                        onChange={(e) => onEmailFilterChange(e.target.value)}
                        className="bg-muted/50 h-9 border-none pl-8 text-sm"
                    />
                </div>

                {/* Status filter */}
                <div className="w-32">
                    <Select
                        value={statusFilter}
                        onValueChange={(val) => onStatusFilterChange(val as InvitationStatus | "")}
                    >
                        <SelectTrigger className="bg-muted/50 h-9 text-sm">
                            <SelectValue placeholder="Status" />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem value="">All Statuses</SelectItem>
                            <SelectItem value="pending">Pending</SelectItem>
                            <SelectItem value="accepted">Accepted</SelectItem>
                            <SelectItem value="expired">Expired</SelectItem>
                            <SelectItem value="revoked">Revoked</SelectItem>
                        </SelectContent>
                    </Select>
                </div>

                {/* Role filter */}
                <div className="w-32">
                    <Select
                        value={roleFilter}
                        onValueChange={(val) => onRoleFilterChange(val as InvitationRole | "")}
                    >
                        <SelectTrigger className="bg-muted/50 h-9 text-sm">
                            <SelectValue placeholder="Role" />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem value="">All Roles</SelectItem>
                            <SelectItem value="SCHOOL_ADMIN">School Admin</SelectItem>
                            <SelectItem value="TEACHER">Teacher</SelectItem>
                            <SelectItem value="SUPPORT_STAFF">Staff</SelectItem>
                        </SelectContent>
                    </Select>
                </div>

                {/* Expired toggle */}
                <Button
                    variant={showExpired ? "outline" : "ghost"}
                    size="sm"
                    onClick={() => onShowExpiredChange(!showExpired)}
                    className={cn("h-9 text-xs", showExpired && "bg-muted/50")}
                >
                    {showExpired ? "Show Expired" : "Hide Expired"}
                    {showExpired && (
                        <X
                            className="ml-1 size-3"
                            onClick={(e) => {
                                e.stopPropagation();
                                onShowExpiredChange(false);
                            }}
                        />
                    )}
                </Button>

                {/* Clear filters */}
                {hasActiveFilters && (
                    <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => {
                            onSearchChange("");
                            onEmailFilterChange("");
                            onStatusFilterChange("");
                            onRoleFilterChange("");
                            onShowExpiredChange(true);
                        }}
                        className="text-muted-foreground h-9 text-xs"
                    >
                        Clear filters
                    </Button>
                )}

                <div className="ml-auto flex items-center gap-2">
                    <Button size="sm" onClick={onInviteClick}>
                        <Mail className="mr-1.5 size-3.5" />
                        Invite Users
                    </Button>
                </div>
            </div>

            {/* Table Canvas */}
            <div
                ref={parentRef}
                className="flex-1 overflow-auto px-6"
                style={{ contain: "strict" }}
            >
                <div className="min-w-[700px]">
                    {/* Sticky Header */}
                    <div className="bg-background/95 sticky top-0 z-10 backdrop-blur-sm">
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
                                    <Skeleton className="mr-3 h-4 w-24" />
                                    <Skeleton className="mr-3 h-4 w-28" />
                                    <Skeleton className="mr-3 h-4 w-16" />
                                    <Skeleton className="mr-3 h-4 w-16" />
                                    <Skeleton className="mr-3 h-4 w-20" />
                                    <Skeleton className="h-4 w-20" />
                                </div>
                            ))
                        ) : rows.length === 0 ? (
                            <div className="flex items-center justify-center py-16">
                                <div className="text-center">
                                    <p className="text-muted-foreground text-sm font-medium">
                                        No invitations found
                                    </p>
                                    <p className="text-muted-foreground mt-1 text-xs">
                                        {hasActiveFilters
                                            ? "Try adjusting your filters."
                                            : "Invite users to get started."}
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
            <div className="border-border/40 flex items-center justify-between border-t px-6 py-2">
                <p className="text-muted-foreground text-xs">
                    {total} invitation{total !== 1 ? "s" : ""}
                </p>
            </div>
        </div>
    );
}
