/**
 * Curriculum page — learning areas listing for the active school.
 *
 * Shows a table of learning areas with filters by education level.
 * Click a row to navigate to the tree view.
 */

"use client";

import Link from "next/link";

import { DataTable, type DataTableColumn } from "@/components/shared/data-table";
import type { FilterGroup } from "@/components/shared/data-table/types";
import { useDeleteLearningArea, curriculumKeys } from "@/features/curriculum";
import { listLearningAreas, type LearningArea } from "@/lib/api/curriculum";

// ─── Education Level Labels ───────────────────────────────────────────────

const EDUCATION_LEVEL_LABELS: Record<string, string> = {
    Early_Years: "Early Years",
    Upper_Primary: "Upper Primary",
    Junior_Secondary: "Junior Secondary",
    Senior_School: "Senior School",
};

function formatEducationLevel(level: string): string {
    return EDUCATION_LEVEL_LABELS[level] ?? level;
}

// ─── Grade Level Labels ───────────────────────────────────────────────────

const GRADE_LEVEL_LABELS: Record<string, string> = {
    PP1: "PP1",
    PP2: "PP2",
    G1: "Grade 1",
    G2: "Grade 2",
    G3: "Grade 3",
    G4: "Grade 4",
    G5: "Grade 5",
    G6: "Grade 6",
    G7: "Grade 7",
    G8: "Grade 8",
    G9: "Grade 9",
    G10: "Grade 10",
    G11: "Grade 11",
    G12: "Grade 12",
};

// ─── Filter Groups ────────────────────────────────────────────────────────

const filterGroups: FilterGroup[] = [
    {
        id: "curriculum_filters",
        label: "Curriculum Filters",
        items: [
            {
                id: "education_level",
                label: "Education Level",
                type: "sub_menu_multi",
                submenu: [
                    { id: "early_years", label: "Early Years", value: "Early_Years" },
                    { id: "upper_primary", label: "Upper Primary", value: "Upper_Primary" },
                    {
                        id: "junior_secondary",
                        label: "Junior Secondary",
                        value: "Junior_Secondary",
                    },
                    { id: "senior_school", label: "Senior School", value: "Senior_School" },
                ],
            },
            {
                id: "grade_level",
                label: "Grade",
                type: "sub_menu_multi",
                submenu: [
                    { id: "pp1", label: "PP1", value: "PP1" },
                    { id: "pp2", label: "PP2", value: "PP2" },
                    { id: "g1", label: "Grade 1", value: "G1" },
                    { id: "g2", label: "Grade 2", value: "G2" },
                    { id: "g3", label: "Grade 3", value: "G3" },
                    { id: "g4", label: "Grade 4", value: "G4" },
                    { id: "g5", label: "Grade 5", value: "G5" },
                    { id: "g6", label: "Grade 6", value: "G6" },
                    { id: "g7", label: "Grade 7", value: "G7" },
                    { id: "g8", label: "Grade 8", value: "G8" },
                    { id: "g9", label: "Grade 9", value: "G9" },
                    { id: "g10", label: "Grade 10", value: "G10" },
                    { id: "g11", label: "Grade 11", value: "G11" },
                    { id: "g12", label: "Grade 12", value: "G12" },
                ],
            },
        ],
    },
];

// ─── Columns ───────────────────────────────────────────────────────────────

const columns: DataTableColumn<LearningArea>[] = [
    {
        id: "name",
        header: "Name",
        cell: (row) => (
            <Link href={`/curriculum/${row.id}`} className="text-sm hover:underline">
                {row.name}
            </Link>
        ),
    },
    {
        id: "code",
        header: "Code",
        width: "120px",
        cell: (row) => (
            <Link
                href={`/curriculum/${row.id}`}
                className="font-mono text-sm font-medium hover:underline"
            >
                {row.code}
            </Link>
        ),
    },
    {
        id: "education_level",
        header: "Education Level",
        width: "180px",
        cell: (row) => (
            <span className="text-muted-foreground text-sm">
                {formatEducationLevel(row.education_level)}
            </span>
        ),
    },
    {
        id: "grade_level",
        header: "Grade",
        width: "120px",
        cell: (row) => (
            <span className="text-muted-foreground text-sm">
                {GRADE_LEVEL_LABELS[row.grade_level] ?? row.grade_level}
            </span>
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
            filterGroups={filterGroups}
            isCheckable
            deleteFn={(id) => deleteMutation.mutateAsync(String(id))}
            addHref="/curriculum/new"
            rowHeight={48}
            height={600}
            emptyState="No learning areas yet."
            noResultsState="No learning areas match your search or filters."
        />
    );
}
