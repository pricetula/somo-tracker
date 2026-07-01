/**
 * Blueprints listing page.
 *
 * Shows all assessment blueprints for the school with filters.
 * Maps to GET /api/v1/assessment/blueprints.
 */

"use client";

import * as React from "react";
import { useRouter } from "next/navigation";

import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { Button } from "@/components/ui/button";
import { Plus } from "lucide-react";

import {
    BlueprintsTable,
    useBlueprints,
    useDeleteBlueprint,
    GRADE_LEVELS,
} from "@/features/assessment";

export default function BlueprintsPage() {
    const router = useRouter();
    const [gradeFilter, setGradeFilter] = React.useState("");
    const [typeFilter, setTypeFilter] = React.useState("");

    const {
        data: blueprintsData,
        isLoading: blueprintsLoading,
        isError: blueprintsError,
    } = useBlueprints({
        grade_level: gradeFilter || undefined,
        type: typeFilter || undefined,
    });

    const deleteBlueprint = useDeleteBlueprint();

    const blueprints = blueprintsData?.data ?? [];

    const handleDelete = async (id: string) => {
        if (window.confirm("Delete this blueprint? This action cannot be undone.")) {
            deleteBlueprint.mutate(id);
        }
    };

    return (
        <div className="flex flex-1 flex-col">
            {/* Page header */}
            <div className="flex items-center gap-3 px-6 pt-6 pb-2">
                <h1 className="text-2xl font-semibold tracking-tight">Assessment Blueprints</h1>
                <div className="ml-auto flex items-center gap-2">
                    <Select
                        value={typeFilter}
                        onValueChange={(v) => setTypeFilter(v === "all" ? "" : v)}
                    >
                        <SelectTrigger className="h-8 w-40 text-xs">
                            <SelectValue placeholder="All types" />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem value="all">All types</SelectItem>
                            <SelectItem value="Formative_Classroom">Formative Classroom</SelectItem>
                            <SelectItem value="KNEC_Written_Assessment">KNEC Written</SelectItem>
                            <SelectItem value="KNEC_SBA_Project">KNEC SBA Project</SelectItem>
                            <SelectItem value="National_KPSEA">National KPSEA</SelectItem>
                            <SelectItem value="National_KJSEA">National KJSEA</SelectItem>
                            <SelectItem value="National_KSSEA">National KSSEA</SelectItem>
                        </SelectContent>
                    </Select>
                    <Select
                        value={gradeFilter}
                        onValueChange={(v) => setGradeFilter(v === "all" ? "" : v)}
                    >
                        <SelectTrigger className="h-8 w-32 text-xs">
                            <SelectValue placeholder="All grades" />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem value="all">All grades</SelectItem>
                            {GRADE_LEVELS.map((g) => (
                                <SelectItem key={g} value={g}>
                                    {g}
                                </SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                    <Button
                        variant="outline"
                        size="sm"
                        onClick={() => router.push("/assessment/blueprints/new")}
                    >
                        <Plus className="mr-1.5 size-3.5" />
                        New Blueprint
                    </Button>
                </div>
            </div>

            <div className="flex flex-1 flex-col px-6 py-4">
                <section className="flex flex-1 flex-col">
                    {blueprintsError ? (
                        <div className="flex items-center justify-center py-8">
                            <p className="text-destructive text-sm">
                                Failed to load blueprints. Please try again.
                            </p>
                        </div>
                    ) : (
                        <div className="ring-foreground/10 rounded-lg ring-1">
                            <BlueprintsTable
                                blueprints={blueprints}
                                total={blueprints.length}
                                isLoading={blueprintsLoading}
                                onDelete={handleDelete}
                                onCreateClick={() => router.push("/assessment/blueprints/new")}
                            />
                        </div>
                    )}
                </section>
            </div>
        </div>
    );
}
