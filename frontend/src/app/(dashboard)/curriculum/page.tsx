/**
 * Curriculum page — learning areas listing for the active school.
 *
 * Shows a table of learning areas with filters by education level and grade.
 * Click a row to navigate to the tree view.
 */

"use client";

import Link from "next/link";

import { DataTable, type DataTableColumn } from "@/components/shared/data-table";
import { useDeleteLearningArea, curriculumKeys } from "@/features/curriculum";
import { listLearningAreas, type LearningArea } from "@/lib/api/curriculum";
import {
    formatEducationLevel,
    formatGradeLevel,
    CURRICULUM_FILTER_GROUPS,
} from "@/lib/curriculum-filters";

// ─── Columns ───────────────────────────────────────────────────────────────

const columns: DataTableColumn<LearningArea>[] = [
    {
        id: "name",
        header: "Name",
        cell: (row) => (
            <Link href={`/curriculum/${row.id}`} className="hover:underline">
                {row.name}
            </Link>
        ),
    },
    {
        id: "code",
        header: "Code",
        width: "120px",
        cell: (row) => (
            <Link href={`/curriculum/${row.id}`} className="font-mono font-medium hover:underline">
                {row.code}
            </Link>
        ),
    },
    {
        id: "education_level",
        header: "Education Level",
        width: "180px",
        cell: (row) => (
            <span className="text-muted-foreground">
                {formatEducationLevel(row.education_level)}
            </span>
        ),
    },
    {
        id: "grade_level",
        header: "Grade",
        width: "120px",
        cell: (row) => (
            <span className="text-muted-foreground">{formatGradeLevel(row.grade_level)}</span>
        ),
    },
];

// ─── Page Component ────────────────────────────────────────────────────────

export default function CurriculumPage() {
    const deleteMutation = useDeleteLearningArea();

    return (
        <DataTable
            queryKey={curriculumKeys.learningAreas.list()}
            queryFn={listLearningAreas}
            columns={columns}
            getRowId={(row) => row.id}
            isSearchable
            searchPlaceholder="Search learning areas..."
            filterGroups={CURRICULUM_FILTER_GROUPS}
            isCheckable
            deleteFn={(id) => deleteMutation.mutateAsync(String(id))}
            addHref="/curriculum/new"
            emptyState="No learning areas yet."
            noResultsState="No learning areas match your search or filters."
        />
    );
}
