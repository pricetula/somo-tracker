"use client";

import {
    listTeachers,
    type TeacherMember,
    type ListTeachersParams,
    type ListTeachersResponse,
} from "@/lib/api/teachers";
import Link from "next/link";
import { GraduationCap, BookOpen } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import type { DataTableColumn, FilterGroup } from "@/components/shared/data-table/types";
import { DataTable } from "@/components/shared/data-table/data-table";
import { InvitationCountBadge } from "@/features/invitations";
import { getGradeLevelFilterSubmenu } from "@/features/grade-level";
import { getEducationLevelFilterSubmenu } from "@/features/education-level";
import { RowActions } from "@/components/shared/data-table/row-actions";
import { useDeleteTeacher } from "@/features/teachers";

// ─── Teacher Role Labels ──────────────────────────────────────────────────

const TEACHER_ROLE_LABELS: Record<string, string> = {
    PRIMARY_CLASS_TEACHER: "Primary Class Teacher",
    SUBJECT_TEACHER: "Subject Teacher",
    SUBSTITUTE_TEACHER: "Substitute Teacher",
};

function formatTeacherRole(role: string | null): string {
    if (!role) return "—";
    return TEACHER_ROLE_LABELS[role] ?? role;
}

// ─── Columns factory ──────────────────────────────────────────────────────

function createColumns(
    deleteMutation: ReturnType<typeof useDeleteTeacher>
): DataTableColumn<TeacherMember>[] {
    return [
        {
            id: "full_name",
            header: "Full Name",
            cell: (row) => (
                <Link href={`/teachers/${row.id}`} className="font-medium hover:underline">
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
            id: "tsc_number",
            header: "TSC Number",
            cell: (row) => (
                <span className="text-muted-foreground font-mono">{row.tsc_number ?? "—"}</span>
            ),
        },
        {
            id: "knec_panel_assessor_id",
            header: "KNEC Panel Assessor ID",
            cell: (row) => (
                <span className="text-muted-foreground font-mono">
                    {row.knec_panel_assessor_id ?? "—"}
                </span>
            ),
        },
        {
            id: "teacher_role",
            header: "Core Assignment Role",
            cell: (row) => (
                <span className="text-muted-foreground">{formatTeacherRole(row.teacher_role)}</span>
            ),
        },
        {
            id: "is_active",
            header: "Account Status",
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

export function TeachersPage() {
    const deleteMutation = useDeleteTeacher();
    const columns = createColumns(deleteMutation);

    return (
        <DataTable<TeacherMember, ListTeachersParams, ListTeachersResponse>
            isCheckable
            addHref="/teachers/add"
            queryKey={["teachers"]}
            queryFn={listTeachers}
            columns={columns}
            getRowId={(row) => row.id}
            isSearchable
            searchPlaceholder="Search by name, email, or TSC number…"
            filterGroups={filterGroups}
            deleteFn={(id) => deleteMutation.mutateAsync(String(id))}
            emptyState="No teachers yet."
            noResultsState="No teachers match your search or filters."
            renderToolBarComponents={() => (
                <InvitationCountBadge
                    key="invitation-count"
                    role="TEACHER"
                    href="/teachers/invitations"
                />
            )}
        />
    );
}
