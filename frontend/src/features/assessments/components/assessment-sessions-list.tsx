/**
 * AssessmentSessionsList — Teacher/admin view listing all assessment sessions.
 *
 * Uses the shared DataTable component with status and evaluation method filters.
 */

"use client";

import Link from "next/link";
import { FileCheck, FileSpreadsheet } from "lucide-react";

import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import type { FilterGroup } from "@/components/shared/data-table/types";

import { listSessions, type AssessmentSession } from "@/lib/api/assessments";
import { useDeleteSession } from "../hooks/use-assessments";
import { StatusBadge } from "./status-badge";
import { EVALUATION_METHOD_LABELS } from "../types";
import { GradeLevelPill } from "@/features/grade-level";

// ─── Columns ──────────────────────────────────────────────────────────────

const columns: DataTableColumn<AssessmentSession>[] = [
    {
        id: "name",
        header: "Assessment",
        cell: (row) => (
            <Link href={`/assessments/${row.id}`} className="font-medium hover:underline">
                {row.name}
            </Link>
        ),
    },
    {
        id: "class_name",
        header: "Class",
        width: "130px",
        align: "right",
        cell: (row) => (
            <Link href={`/classes/${row.class_id}`} className="font-medium hover:underline">
                {row.class_name}
            </Link>
        ),
    },
    {
        id: "grade_level",
        header: "Grade",
        width: "130px",
        cell: (row) => <GradeLevelPill grade={row.grade_level} />,
    },
    {
        id: "status",
        header: "Status",
        width: "130px",
        cell: (row) => <StatusBadge status={row.status} />,
    },
    {
        id: "max_points",
        header: "Max",
        width: "130px",
        align: "right",
        cell: (row) => (
            <span className="text-muted-foreground tabular-nums">{row.max_points ?? "-"}</span>
        ),
    },
    {
        id: "scheduled_date",
        header: "Scheduled Date",
        width: "130px",
        cell: (row) => <span className="text-muted-foreground">{row.scheduled_date ?? "-"}</span>,
    },
    {
        id: "evaluation_method",
        header: "Type",
        width: "180px",
        cell: (row) => {
            const isQuant = row.evaluation_method === "QUANTITATIVE";
            return (
                <div className="flex items-center gap-1.5">
                    {isQuant ? (
                        <FileSpreadsheet className="text-muted-foreground h-3.5 w-3.5" />
                    ) : (
                        <FileCheck className="text-muted-foreground h-3.5 w-3.5" />
                    )}
                    <span className="text-muted-foreground text-xs">
                        {EVALUATION_METHOD_LABELS[row.evaluation_method] ?? row.evaluation_method}
                    </span>
                </div>
            );
        },
    },
];

// ─── Filter Groups ────────────────────────────────────────────────────────

const filterGroups: FilterGroup[] = [
    {
        id: "status_group",
        label: "Filter by",
        items: [
            {
                id: "status",
                label: "Status",
                type: "sub_menu_multi",
                submenu: [
                    { id: "draft", label: "Draft", value: "DRAFT" },
                    { id: "pending", label: "Pending Approval", value: "PENDING_APPROVAL" },
                    { id: "published", label: "Published", value: "PUBLISHED" },
                ],
            },
            {
                id: "evaluation_method",
                label: "Type",
                type: "sub_menu_multi",
                submenu: [
                    { id: "quant", label: "Marks-Based", value: "QUANTITATIVE" },
                    { id: "rubric", label: "Rubric", value: "RUBRIC" },
                ],
            },
        ],
    },
];

// ─── Component ────────────────────────────────────────────────────────────

export function AssessmentSessionsList() {
    const deleteMutation = useDeleteSession();

    return (
        <DataTable
            isCheckable
            isRowCheckable={(row) => row.status === "DRAFT"}
            addHref="/assessments/add"
            queryKey={["assessment-sessions"]}
            queryFn={listSessions}
            columns={columns}
            getRowId={(row) => row.id}
            isSearchable
            searchPlaceholder="Search by assessment name..."
            filterGroups={filterGroups}
            deleteFn={(id) => deleteMutation.mutateAsync(String(id))}
            emptyState="No assessment sessions yet."
            noResultsState="No sessions match your search or filters."
        />
    );
}
