/**
 * Curriculum page — learning areas listing for the active school.
 *
 * Shows a table of learning areas with filters by education level and grade.
 * Click a row to navigate to the tree view.
 */

"use client";

import Link from "next/link";

import { GraduationCap, LayersPlus, BookOpen } from "lucide-react";
import { DataTable, type DataTableColumn } from "@/components/shared/data-table";
import type { FilterGroup } from "@/components/shared/data-table/types";
import { useDeleteLearningArea, curriculumKeys } from "@/features/curriculum";
import { listLearningAreas, type LearningArea } from "@/lib/api/curriculum";
import { getEducationLevelFilterSubmenu } from "@/features/education-level";
import { getGradeLevelFilterSubmenu } from "@/features/grade-level";
import { EducationLevelPill } from "@/features/education-level";
import { GradeLevelPill } from "@/features/grade-level";
import { useSeedSchool } from "@/features/school/hooks/use-schools";
import { Button } from "@/components/ui/button";

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
        cell: (row) => <EducationLevelPill level={row.education_level} />,
    },
    {
        id: "grade_level",
        header: "Grade",
        width: "120px",
        cell: (row) => <GradeLevelPill grade={row.grade_level} />,
    },
];

// ─── Toolbar ──────────────────────────────────────────────────────────────

function ToolBar() {
    const seedSchoolMutation = useSeedSchool();

    return (
        <Button size="sm" id="enroll-selected-students" onClick={() => seedSchoolMutation.mutate()}>
            <LayersPlus />
            <span>Set default Curriculum</span>
        </Button>
    );
}

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
            emptyState="No learning areas yet."
            noResultsState="No learning areas match your search or filters."
            renderToolBarComponents={() => <ToolBar />}
        />
    );
}
