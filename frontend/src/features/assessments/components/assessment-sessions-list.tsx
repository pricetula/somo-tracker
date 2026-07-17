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
import { StatusBadge } from "./status-badge";
import { EVALUATION_METHOD_LABELS } from "../types";

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
        id: "status",
        header: "Status",
        width: "140px",
        cell: (row) => <StatusBadge status={row.status} />,
    },
    {
        id: "evaluation_method",
        header: "Type",
        width: "130px",
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
    {
        id: "max_points",
        header: "Max",
        width: "60px",
        align: "right",
        cell: (row) => (
            <span className="text-muted-foreground tabular-nums">{row.max_points ?? "\u2014"}</span>
        ),
    },
    {
        id: "scheduled_date",
        header: "Date",
        width: "110px",
        cell: (row) => (
            <span className="text-muted-foreground">{row.scheduled_date ?? "\u2014"}</span>
        ),
    },
    {
        id: "created_at",
        header: "Created",
        width: "110px",
        cell: (row) => (
            <span className="text-muted-foreground text-xs">
                {new Date(row.created_at).toLocaleDateString()}
            </span>
        ),
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
    return (
        <DataTable
            addHref="/assessments/add"
            queryKey={["assessment-sessions"]}
            queryFn={listSessions}
            columns={columns}
            getRowId={(row) => row.id}
            isSearchable
            searchPlaceholder="Search by assessment name..."
            filterGroups={filterGroups}
            emptyState="No assessment sessions yet."
            noResultsState="No sessions match your search or filters."
        />
    );
}
