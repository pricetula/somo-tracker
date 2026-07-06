/**
 * Curriculum page — learning areas listing for the active school.
 *
 * Shows a table of learning areas with filters by education level.
 * Click a row to navigate to the tree view.
 */

"use client";

import * as React from "react";
import Link from "next/link";

import { DataTable, type DataTableColumn } from "@/components/shared/data-table";
import {
    CreateLearningAreaDialog,
    useDeleteLearningArea,
    curriculumKeys,
} from "@/features/curriculum";
import { listLearningAreas, type LearningArea } from "@/lib/api/curriculum";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { Button } from "@/components/ui/button";
import { Plus } from "lucide-react";

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

// ─── Columns ───────────────────────────────────────────────────────────────

const columns: DataTableColumn<LearningArea>[] = [
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
        id: "name",
        header: "Name",
        cell: (row) => (
            <Link href={`/curriculum/${row.id}`} className="text-sm hover:underline">
                {row.name}
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
];

// ─── Page Component ────────────────────────────────────────────────────────

export default function CurriculumPage() {
    const [educationLevel, setEducationLevel] = React.useState<string>("");
    const [createOpen, setCreateOpen] = React.useState(false);

    const deleteMutation = useDeleteLearningArea();

    const params = React.useMemo(
        () => ({ education_level: educationLevel || undefined }),
        [educationLevel]
    );

    return (
        <div className="flex flex-1 flex-col">
            {/* Page header */}
            <div className="flex items-center gap-3 px-6 pt-6 pb-2">
                <h1 className="text-2xl font-semibold tracking-tight">Curriculum</h1>
                <div className="ml-auto flex items-center gap-2">
                    <Select
                        value={educationLevel}
                        onValueChange={(v) => setEducationLevel(v === "all" ? "" : v)}
                    >
                        <SelectTrigger className="h-8 w-44 text-xs">
                            <SelectValue placeholder="All levels" />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem value="all">All levels</SelectItem>
                            <SelectItem value="Early_Years">Early Years</SelectItem>
                            <SelectItem value="Upper_Primary">Upper Primary</SelectItem>
                            <SelectItem value="Junior_Secondary">Junior Secondary</SelectItem>
                            <SelectItem value="Senior_School">Senior School</SelectItem>
                        </SelectContent>
                    </Select>
                    <Button variant="outline" size="sm" onClick={() => setCreateOpen(true)}>
                        <Plus className="mr-1.5 size-3.5" />
                        Add Learning Area
                    </Button>
                </div>
            </div>

            <div className="flex flex-1 flex-col px-6 py-4">
                <section className="flex flex-1 flex-col">
                    <DataTable
                        queryKey={curriculumKeys.learningAreas.list(params)}
                        queryFn={listLearningAreas}
                        params={params}
                        columns={columns}
                        getRowId={(row) => row.id}
                        isSearchable
                        searchPlaceholder="Search learning areas..."
                        isCheckable
                        deleteFn={(id) => deleteMutation.mutateAsync(String(id))}
                        rowHeight={48}
                        height={600}
                        emptyState="No learning areas yet."
                        noResultsState="No learning areas match your search."
                    />
                </section>
            </div>

            {/* Create Learning Area Dialog */}
            <CreateLearningAreaDialog open={createOpen} onOpenChange={setCreateOpen} />
        </div>
    );
}
