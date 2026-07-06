/**
 * Teachers listing page — active teachers with extended educator fields.
 *
 * Uses the shared DataTable component with curriculum filter.
 * Maps to GET /api/v1/teachers.
 *
 * Invitations are listed on the dedicated /teachers/invitations page.
 */

"use client";

import Link from "next/link";
import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Upload } from "lucide-react";
import { listTeachers, type TeacherMember } from "@/lib/api/teachers";
import { CURRICULUM_FILTER_GROUPS } from "@/lib/curriculum-filters";
import { useDeleteTeacher } from "@/features/staff";

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
        cell: (row) => <span className="text-sm font-medium">{row.full_name || "—"}</span>,
    },
    {
        id: "email",
        header: "Email",
        cell: (row) => <span className="text-muted-foreground text-sm">{row.email}</span>,
    },
    {
        id: "tsc_number",
        header: "TSC Number",
        cell: (row) => (
            <span className="text-muted-foreground font-mono text-sm">{row.tsc_number ?? "—"}</span>
        ),
    },
    {
        id: "knec_panel_assessor_id",
        header: "KNEC Panel Assessor ID",
        cell: (row) => (
            <span className="text-muted-foreground font-mono text-sm">
                {row.knec_panel_assessor_id ?? "—"}
            </span>
        ),
    },
    {
        id: "teacher_role",
        header: "Core Assignment Role",
        cell: (row) => (
            <span className="text-muted-foreground text-sm">
                {formatTeacherRole(row.teacher_role)}
            </span>
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

// ─── Page ─────────────────────────────────────────────────────────────────

export default function TeachersPage() {
    const deleteMutation = useDeleteTeacher();

    return (
        <div className="flex flex-1 flex-col gap-4">
            <div className="flex items-center justify-between">
                <h1 className="text-2xl font-semibold tracking-tight">Teachers</h1>
                <Button variant="outline" size="sm" asChild>
                    <Link href="/teachers/import">
                        <Upload className="mr-1.5 size-3.5" />
                        Bulk Import
                    </Link>
                </Button>
            </div>
            <DataTable
                queryKey={["teachers"]}
                queryFn={listTeachers}
                columns={columns}
                getRowId={(row) => row.id}
                isSearchable
                searchPlaceholder="Search by name, email, or TSC number…"
                filterGroups={CURRICULUM_FILTER_GROUPS}
                deleteFn={(id) => deleteMutation.mutateAsync(String(id))}
                rowHeight={48}
                height={600}
                emptyState="No teachers yet."
                noResultsState="No teachers match your search or filters."
            />
        </div>
    );
}
