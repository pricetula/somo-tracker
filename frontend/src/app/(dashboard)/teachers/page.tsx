/**
 * Teachers listing page — active teachers with extended educator fields.
 *
 * Uses the shared DataTable component with curriculum filter.
 * Maps to GET /api/v1/teachers.
 *
 * Invitations are listed on the dedicated /teachers/invitations page.
 */

"use client";

import { useQuery } from "@tanstack/react-query";
import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn, FilterGroup } from "@/components/shared/data-table/types";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { listTeachers, type TeacherMember } from "@/lib/api/teachers";
import { getInvitationCount } from "@/lib/api/invitations";
import { GraduationCap, BookOpen } from "lucide-react";
import Link from "next/link";
import { getEducationLevelFilterSubmenu } from "@/features/education-level";
import { getGradeLevelFilterSubmenu } from "@/features/grade-level";
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

// ─── Columns ──────────────────────────────────────────────────────────────

const columns: DataTableColumn<TeacherMember>[] = [
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
];

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

// ─── Invitation Count Badge ────────────────────────────────────────────────

function InvitationCountBadge() {
    const { data, isLoading } = useQuery({
        queryKey: ["invitations", "count", "TEACHER"],
        queryFn: () => getInvitationCount("TEACHER"),
    });

    if (isLoading) {
        return <Skeleton className="h-9 w-28" />;
    }

    const count = data?.total ?? 0;
    const label = `${count} ${count === 1 ? "invitation" : "invitations"}`;

    return (
        <Button variant="outline" size="sm" asChild>
            <Link href="/teachers/invitations">{label}</Link>
        </Button>
    );
}

// ─── Page ─────────────────────────────────────────────────────────────────

export default function TeachersPage() {
    const deleteMutation = useDeleteTeacher();

    return (
        <DataTable
            addHref="/teachers/import"
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
            toolBarComponents={[<InvitationCountBadge key="invitation-count" />]}
        />
    );
}
