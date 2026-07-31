/**
 * Students listing page — active enrolled students.
 *
 * Uses the shared DataTable component with curriculum and lifecycle filters,
 * bulk delete, and per-row actions.
 * Maps to GET /api/v1/students/list.
 */

"use client";

import { useCallback, useMemo } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import type { FilterGroup } from "@/components/shared/data-table/types";
import { RowActions } from "@/components/shared/data-table/row-actions";
import { Badge } from "@/components/ui/badge";
import { listStudents, type Student } from "@/lib/api/students";
import { GraduationCap, BookOpen, Pencil } from "lucide-react";
import { getEducationLevelFilterSubmenu } from "@/features/education-level";
import { getGradeLevelFilterSubmenu } from "@/features/grade-level";
import { useDeleteStudent, useEnrollmentStore } from "@/features/students";
import { Button } from "@/components/ui/button";

type AppRouterInstance = ReturnType<typeof useRouter>;

// ─── Filter Groups (curriculum + lifecycle) ───────────────────────────────

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
    {
        id: "lifecycle_group",
        label: "Lifecycle Group",
        items: [
            {
                id: "enrollment_status",
                label: "Enrollment Status",
                type: "sub_menu_single",
                submenu: [
                    { id: "active_status", label: "Active", value: "ACTIVE" },
                    { id: "suspended_status", label: "Suspended", value: "SUSPENDED" },
                    { id: "transferred_status", label: "Transferred", value: "TRANSFERRED" },
                ],
            },
        ],
    },
];

// ─── Columns factory ──────────────────────────────────────────────────────

function createColumns(
    deleteMutation: ReturnType<typeof useDeleteStudent>,
    router: AppRouterInstance
): DataTableColumn<Student>[] {
    return [
        {
            id: "full_name",
            header: "Full Name",
            cell: (row) => (
                <Link href={`/students/${row.id}`} className="font-medium hover:underline">
                    {row.full_name}
                </Link>
            ),
        },
        {
            id: "admission_number",
            header: "Admission No.",
            cell: (row) => (
                <span className="text-muted-foreground font-mono">
                    {row.admission_number ?? "—"}
                </span>
            ),
        },
        {
            id: "upi_number",
            header: "UPI Number",
            cell: (row) => (
                <span className="text-muted-foreground font-mono">{row.upi_number ?? "—"}</span>
            ),
        },
        {
            id: "knec_assessment_number",
            header: "KNEC No.",
            cell: (row) => (
                <span className="text-muted-foreground font-mono">
                    {row.knec_assessment_number ?? "—"}
                </span>
            ),
        },
        {
            id: "class_name",
            header: "Class",
            cell: (row) => <span className="text-muted-foreground">{row.class_name ?? "—"}</span>,
        },
        {
            id: "gender",
            header: "Gender",
            width: "80px",
            cell: (row) => (
                <span className="text-muted-foreground">
                    {row.gender === "M" ? "Male" : row.gender === "F" ? "Female" : "—"}
                </span>
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
                    label={row.full_name}
                    onDelete={() => deleteMutation.mutate(row.id)}
                    disabled={deleteMutation.isPending}
                    actions={[
                        {
                            label: "Edit",
                            icon: Pencil,
                            onClick: () => router.push(`/students/${row.id}`),
                        },
                    ]}
                />
            ),
        },
    ];
}

// ─── Toolbar ──────────────────────────────────────────────────────────────

function ToolBar({ selectedIds }: { selectedIds: Set<string> }) {
    const router = useRouter();
    const setSelectedStudentIds = useEnrollmentStore((s) => s.setSelectedStudentIds);
    const handleEnroll = useCallback(() => {
        setSelectedStudentIds([...selectedIds]);
        router.push("/students/enroll");
    }, [selectedIds, router, setSelectedStudentIds]);

    if (selectedIds.size === 0) return null;

    return (
        <Button size="sm" id="enroll-selected-students" onClick={handleEnroll}>
            <GraduationCap />
            <span>Enroll {`${selectedIds.size} Student${selectedIds.size === 1 ? "" : "s"}`}</span>
        </Button>
    );
}

// ─── Page ─────────────────────────────────────────────────────────────────

export default function StudentsPage() {
    const router = useRouter();
    const deleteMutation = useDeleteStudent();
    const columns = useMemo(() => createColumns(deleteMutation, router), [deleteMutation, router]);

    return (
        <DataTable
            isCheckable
            addHref="/students/import"
            queryKey={["students"]}
            queryFn={listStudents}
            columns={columns}
            getRowId={(row) => row.id}
            isSearchable
            searchPlaceholder="Search by name, admission no., UPI, or KNEC no…"
            filterGroups={filterGroups}
            deleteFn={(id) => deleteMutation.mutateAsync(String(id))}
            emptyState="No students yet."
            noResultsState="No students match your search or filters."
            renderToolBarComponents={(s: Set<string>) => <ToolBar selectedIds={s} />}
        />
    );
}
