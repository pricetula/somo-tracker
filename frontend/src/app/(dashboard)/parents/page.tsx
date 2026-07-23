/**
 * Parents listing page.
 *
 * Shows all parent/guardian profiles with search and curriculum filter.
 * Uses the shared DataTable component with bulk delete and per-row actions.
 * Maps to GET /api/v1/parents.
 */

"use client";

import Link from "next/link";
import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn, FilterGroup } from "@/components/shared/data-table/types";
import { RowActions } from "@/components/shared/data-table/row-actions";
import { Badge } from "@/components/ui/badge";
import { GraduationCap, BookOpen } from "lucide-react";
import { listParents, type Parent } from "@/lib/api/parents";
import { getEducationLevelFilterSubmenu } from "@/features/education-level";
import { getGradeLevelFilterSubmenu } from "@/features/grade-level";
import { InvitationCountBadge } from "@/features/invitations";
import { useDeleteParent } from "@/features/parents";

// ─── Columns factory ──────────────────────────────────────────────────────

function createColumns(
    deleteMutation: ReturnType<typeof useDeleteParent>
): DataTableColumn<Parent>[] {
    return [
        {
            id: "full_name",
            header: "Full Name",
            cell: (row) => (
                <Link href={`/parents/${row.id}`} className="font-medium hover:underline">
                    {row.full_name || "—"}
                </Link>
            ),
        },
        {
            id: "email",
            header: "Email",
            cell: (row) => <span className="text-muted-foreground">{row.email}</span>,
        },
        {
            id: "phone_number",
            header: "Phone",
            cell: (row) => (
                <span className="text-muted-foreground font-mono">{row.phone_number || "—"}</span>
            ),
        },
        {
            id: "is_active",
            header: "Status",
            width: "100px",
            cell: (row) => (
                <Badge
                    variant="secondary"
                    className={
                        row.is_active
                            ? "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400"
                            : "bg-muted text-muted-foreground"
                    }
                >
                    {row.is_active ? "Active" : "Inactive"}
                </Badge>
            ),
        },
        {
            id: "actions",
            header: "",
            width: "48px",
            align: "right",
            cell: (row) => (
                <RowActions
                    rowId={row.id}
                    label={row.full_name ?? row.email}
                    onDelete={() => deleteMutation.mutate(row.id)}
                    disabled={deleteMutation.isPending}
                />
            ),
        },
    ];
}

// ─── Filter Groups ────────────────────────────────────────────────────────

const filterGroups: FilterGroup[] = [
    {
        id: "curriculum_filters",
        label: "Filter by",
        items: [
            {
                id: "education_level",
                label: "Education Level",
                icon: BookOpen,
                type: "sub_menu_multi",
                submenu: getEducationLevelFilterSubmenu(),
            },
            {
                id: "grade_level",
                label: "Grade",
                icon: GraduationCap,
                type: "sub_menu_multi",
                submenu: getGradeLevelFilterSubmenu(),
            },
        ],
    },
];

// ─── Page ─────────────────────────────────────────────────────────────────

export default function ParentsPage() {
    const deleteMutation = useDeleteParent();
    const columns = createColumns(deleteMutation);

    return (
        <DataTable
            isCheckable
            addHref="/parents/import"
            queryKey={["parents"]}
            queryFn={listParents}
            columns={columns}
            getRowId={(row) => row.id}
            isSearchable
            searchPlaceholder="Search by name or email…"
            filterGroups={filterGroups}
            deleteFn={(id) => deleteMutation.mutateAsync(String(id))}
            emptyState="No parents yet."
            noResultsState="No parents match your search or filters."
            renderToolBarComponents={() => (
                <InvitationCountBadge
                    key="invitation-count"
                    role="PARENT"
                    href="/parents/invitations"
                />
            )}
        />
    );
}
