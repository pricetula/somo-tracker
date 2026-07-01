/**
 * Curriculum page — learning areas listing for the active school.
 *
 * Shows a table of learning areas with filters by education level.
 * Click a row to navigate to the tree view.
 */

"use client";

import * as React from "react";

import {
    LearningAreasTable,
    CreateLearningAreaDialog,
    useLearningAreas,
} from "@/features/curriculum";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { Button } from "@/components/ui/button";
import { Plus } from "lucide-react";

export default function CurriculumPage() {
    const [educationLevel, setEducationLevel] = React.useState<string>("");
    const [createOpen, setCreateOpen] = React.useState(false);

    const {
        data: areasData,
        isLoading: areasLoading,
        isError: areasError,
    } = useLearningAreas({ education_level: educationLevel || undefined });

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
                    {areasError ? (
                        <div className="flex items-center justify-center py-8">
                            <p className="text-destructive text-sm">
                                Failed to load curriculum. Please try again.
                            </p>
                        </div>
                    ) : (
                        <div className="ring-foreground/10 rounded-lg ring-1">
                            <LearningAreasTable
                                learningAreas={areasData?.learning_areas ?? []}
                                total={areasData?.total ?? 0}
                                isLoading={areasLoading}
                                onCreateClick={() => setCreateOpen(true)}
                            />
                        </div>
                    )}
                </section>
            </div>

            {/* Create Learning Area Dialog */}
            <CreateLearningAreaDialog open={createOpen} onOpenChange={setCreateOpen} />
        </div>
    );
}
