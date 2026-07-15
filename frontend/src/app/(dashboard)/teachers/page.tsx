/**
 * Teachers listing page — active teachers with extended educator fields.
 *
 * Uses the shared DataTable component with curriculum filter.
 * Maps to GET /api/v1/teachers.
 *
 * Invitations are listed on the dedicated /teachers/invitations page.
 */

"use client";

import { useState } from "react";
import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn, FilterGroup } from "@/components/shared/data-table/types";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { listTeachers, type TeacherMember } from "@/lib/api/teachers";
import { GraduationCap, BookOpen, Pencil } from "lucide-react";
import { getEducationLevelFilterSubmenu } from "@/features/education-level";
import { getGradeLevelFilterSubmenu } from "@/features/grade-level";
import { useDeleteTeacher } from "@/features/staff";
import { TeacherEditDialog } from "@/features/staff/components/teacher-edit-dialog";

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
        cell: (row) => <span className="font-medium">{row.full_name || "—"}</span>,
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

// ─── Edit action cell component ──────────────────────────────────────────

function EditCell({ row }: { row: TeacherMember }) {
    const [editOpen, setEditOpen] = useState(false);
    return (
        <>
            <Button
                variant="ghost"
                size="icon-sm"
                onClick={() => setEditOpen(true)}
                title="Edit teacher"
            >
                <Pencil className="h-4 w-4" />
                <span className="sr-only">Edit {row.full_name}</span>
            </Button>
            <TeacherEditDialog userId={row.id} open={editOpen} onOpenChange={setEditOpen} />
        </>
    );
}

const editActionColumn: DataTableColumn<TeacherMember> = {
    id: "edit",
    header: "",
    width: "48px",
    align: "right",
    cell: (row) => <EditCell row={row} />,
};

// ─── Page ─────────────────────────────────────────────────────────────────

export default function TeachersPage() {
    const deleteMutation = useDeleteTeacher();

    return (
        <DataTable
            addHref="/teachers/import"
            queryKey={["teachers"]}
            queryFn={listTeachers}
            columns={[...columns, editActionColumn]}
            getRowId={(row) => row.id}
            isSearchable
            searchPlaceholder="Search by name, email, or TSC number…"
            filterGroups={filterGroups}
            deleteFn={(id) => deleteMutation.mutateAsync(String(id))}
            emptyState="No teachers yet."
            noResultsState="No teachers match your search or filters."
        />
    );
}
