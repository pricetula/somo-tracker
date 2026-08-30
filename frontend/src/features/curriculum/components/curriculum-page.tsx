/**
 * Curriculum index page — feature-level container using DataTable.
 */

"use client";

import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import { GradeLevelPill } from "@/features/grade-level";
import { EducationLevelPill } from "@/features/education-level";

import { SeedDefaultButton } from "./seed-default-button";
import { listLearningAreas, type LearningArea } from "@/lib/api/curriculum";

const columns: DataTableColumn<LearningArea>[] = [
    {
        id: "name",
        header: "Name",
        cell: (row) => (
            <a href={`/curriculum/${row.id}`} className="font-medium hover:underline">
                {row.name}
            </a>
        ),
    },
    {
        id: "code",
        header: "Code",
        width: "100px",
        cell: (row) => <span className="font-mono text-xs">{row.code}</span>,
    },
    {
        id: "grade_level",
        header: "Grade",
        width: "80px",
        cell: (row) => <GradeLevelPill grade={row.grade_level} />,
    },
    {
        id: "education_level",
        header: "Level",
        width: "120px",
        cell: (row) => <EducationLevelPill level={row.education_level} />,
    },
];

export function CurriculumPage() {
    return (
        <DataTable
            queryKey={["curriculum", "learning-areas"]}
            queryFn={listLearningAreas}
            columns={columns}
            getRowId={(row) => row.id}
            isSearchable
            searchPlaceholder="Search learning areas..."
            addHref="/curriculum/add"
            emptyState="No learning areas yet."
            noResultsState="No learning areas match your search."
            isCheckable
            renderToolBarComponents={() => <SeedDefaultButton />}
        />
    );
}
